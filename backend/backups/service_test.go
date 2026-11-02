package backups

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/steam"
	"sentinel/backend/steam/types"

	"github.com/stretchr/testify/require"
)

func TestSnapshotRoundTrip(t *testing.T) {
	old := backend.DataDir
	backend.DataDir = t.TempDir()
	t.Cleanup(func() { backend.DataDir = old })
	backupDir := t.TempDir()
	s := &Service{Config: &config.File{BackupDirectory: backupDir}}
	for _, name := range cacheFolders {
		require.NoError(t, os.MkdirAll(filepath.Join(backend.DataDir, name), 0755))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(backend.DataDir, "games", "english"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backend.DataDir, "games", "english", "1.json"), []byte("original"), 0600))
	require.NoError(t, s.Backup())
	require.NoError(t, s.Backup())
	items, err := s.List()
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.NotEqual(t, items[0].Name, items[1].Name)
	archive, err := zip.OpenReader(filepath.Join(backupDir, items[0].Name))
	require.NoError(t, err)
	var names []string
	for _, f := range archive.File {
		names = append(names, f.Name)
	}
	require.NoError(t, archive.Close())
	for _, folder := range cacheFolders {
		require.Contains(t, names, folder+"/")
	}
	require.Contains(t, names, "games/english/1.json")
	require.NoError(t, os.WriteFile(filepath.Join(backend.DataDir, "data", "extra"), []byte("extra"), 0600))
	require.NoError(t, s.Restore(filepath.Join(backupDir, items[0].Name)))
	data, err := os.ReadFile(filepath.Join(backend.DataDir, "games", "english", "1.json"))
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	_, err = os.Stat(filepath.Join(backend.DataDir, "data", "extra"))
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Error(t, s.Delete("../outside.zip"))
	require.NoError(t, s.Delete(items[0].Name))
	newDir := t.TempDir()
	s.Config.BackupDirectory = newDir
	list, err := s.List()
	require.NoError(t, err)
	require.Empty(t, list)
	require.NoError(t, s.Backup())
	list, err = s.List()
	require.NoError(t, err)
	require.Len(t, list, 1)
	_, err = os.Stat(filepath.Join(backupDir, items[1].Name))
	require.NoError(t, err)
}

func TestRejectUnsafeArchiveBeforeReplacement(t *testing.T) {
	old := backend.DataDir
	backend.DataDir = t.TempDir()
	t.Cleanup(func() { backend.DataDir = old })
	require.NoError(t, os.Mkdir(filepath.Join(backend.DataDir, "data"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backend.DataDir, "data", "keep"), []byte("safe"), 0600))
	file, err := os.Create(filepath.Join(t.TempDir(), "arbitrary.zip"))
	require.NoError(t, err)
	z := zip.NewWriter(file)
	for _, name := range []string{"data/", "games/", "icon/", "data/../../escape"} {
		_, err = z.Create(name)
		require.NoError(t, err)
	}
	require.NoError(t, z.Close())
	require.NoError(t, file.Close())
	s := &Service{Config: &config.File{BackupDirectory: t.TempDir()}}
	require.Error(t, s.Restore(file.Name()))
	data, err := os.ReadFile(filepath.Join(backend.DataDir, "data", "keep"))
	require.NoError(t, err)
	require.Equal(t, "safe", string(data))
}

func TestRollbackRestoresPreviousDirectories(t *testing.T) {
	old := backend.DataDir
	backend.DataDir = t.TempDir()
	t.Cleanup(func() { backend.DataDir = old })
	rollback := filepath.Join(backend.DataDir, ".snapshot-rollback")
	require.NoError(t, os.MkdirAll(filepath.Join(rollback, "data"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rollback, "data", "old"), []byte("old"), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(backend.DataDir, "data"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backend.DataDir, "data", "new"), []byte("new"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(rollback, "games.absent"), nil, 0600))
	require.NoError(t, os.Mkdir(filepath.Join(backend.DataDir, "games"), 0755))
	require.Error(t, rollbackFailure(rollback, os.ErrPermission))
	data, err := os.ReadFile(filepath.Join(backend.DataDir, "data", "old"))
	require.NoError(t, err)
	require.Equal(t, "old", string(data))
	_, err = os.Stat(filepath.Join(backend.DataDir, "games"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRejectOverlappingOperations(t *testing.T) {
	s := &Service{}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- s.run("backup", func() error { close(entered); <-release; return nil }) }()
	<-entered
	require.Error(t, s.Restore("unused"))
	close(release)
	require.NoError(t, <-done)
}

func TestReplacementFailureAtEachFolder(t *testing.T) {
	old := backend.DataDir
	t.Cleanup(func() { backend.DataDir = old })
	for failure := range cacheFolders {
		t.Run(cacheFolders[failure], func(t *testing.T) {
			backend.DataDir = t.TempDir()
			stage := filepath.Join(backend.DataDir, "stage")
			require.NoError(t, os.Mkdir(stage, 0755))
			for i, folder := range cacheFolders {
				require.NoError(t, os.Mkdir(filepath.Join(backend.DataDir, folder), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(backend.DataDir, folder, "old"), []byte(folder), 0600))
				if i != failure {
					require.NoError(t, os.Mkdir(filepath.Join(stage, folder), 0755))
				}
			}
			require.Error(t, replace(stage))
			for _, folder := range cacheFolders {
				data, err := os.ReadFile(filepath.Join(backend.DataDir, folder, "old"))
				require.NoError(t, err)
				require.Equal(t, folder, string(data))
			}
		})
	}
}

func TestInvalidSnapshotStructures(t *testing.T) {
	old := backend.DataDir
	backend.DataDir = t.TempDir()
	t.Cleanup(func() { backend.DataDir = old })
	cases := map[string][]string{
		"missing":  {"data/", "games/"},
		"extra":    {"data/", "games/", "icon/", "other/"},
		"absolute": {"data/", "games/", "icon/", "/escape"},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			file, err := os.Create(filepath.Join(t.TempDir(), "import.zip"))
			require.NoError(t, err)
			writer := zip.NewWriter(file)
			for _, entry := range entries {
				_, err := writer.Create(entry)
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			require.NoError(t, file.Close())
			require.Error(t, (&Service{}).Restore(file.Name()))
		})
	}
	malformed := filepath.Join(t.TempDir(), "bad.zip")
	require.NoError(t, os.WriteFile(malformed, []byte("not a zip"), 0600))
	require.Error(t, (&Service{}).Restore(malformed))
	symlinkFile, err := os.Create(filepath.Join(t.TempDir(), "link.zip"))
	require.NoError(t, err)
	writer := zip.NewWriter(symlinkFile)
	for _, entry := range []string{"data/", "games/", "icon/"} {
		_, err := writer.Create(entry)
		require.NoError(t, err)
	}
	header := &zip.FileHeader{Name: "data/link"}
	header.SetMode(os.ModeSymlink | 0777)
	_, err = writer.CreateHeader(header)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, symlinkFile.Close())
	require.Error(t, (&Service{}).Restore(symlinkFile.Name()))
}

func TestRestoreRelocatedGameCache(t *testing.T) {
	oldData, oldGames, oldAch := backend.DataDir, backend.GameCacheDir, backend.ACHCacheDataDir
	t.Cleanup(func() { backend.DataDir, backend.GameCacheDir, backend.ACHCacheDataDir = oldData, oldGames, oldAch })
	backend.DataDir = t.TempDir()
	for _, language := range []string{"english", "french"} {
		dir := filepath.Join(backend.DataDir, "games", language)
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "123.json"), []byte(`{"AppID":"123","Name":"Game"}`), 0600))
	}
	cfg := &config.File{BackupDirectory: t.TempDir(), Language: types.Language{API: "french"}}
	s := &Service{Config: cfg}
	require.NoError(t, s.Backup())
	list, err := s.List()
	require.NoError(t, err)
	require.Len(t, list, 1)
	backend.DataDir = t.TempDir()
	backend.GameCacheDir = filepath.Join(backend.DataDir, "games")
	backend.ACHCacheDataDir = filepath.Join(backend.DataDir, "data")
	require.NoError(t, s.Restore(filepath.Join(cfg.BackupDirectory, list[0].Name)))
	for _, language := range []string{"english", "french"} {
		_, err := os.Stat(filepath.Join(backend.GameCacheDir, language, "123.json"))
		require.NoError(t, err)
	}
	games, err := (&steam.Service{Config: cfg, Ach: &ach.Service{}}).LoadAllCachedGameData()
	require.NoError(t, err)
	require.Len(t, games, 1)
	require.Equal(t, "123", games[0].AppID)
	require.Equal(t, "french", cfg.Language.API)
}

func TestListSortsTimestampsAndRetainsRecovery(t *testing.T) {
	old := backend.DataDir
	backend.DataDir = t.TempDir()
	t.Cleanup(func() { backend.DataDir = old })
	dir := t.TempDir()
	for _, name := range []string{"sentinel-100.zip", "sentinel-200.zip", "unrelated.zip"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0600))
	}
	s := &Service{Config: &config.File{BackupDirectory: dir}}
	items, err := s.List()
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.EqualValues(t, 200, items[0].Timestamp)
	require.EqualValues(t, 100, items[1].Timestamp)
	rollback := filepath.Join(backend.DataDir, ".snapshot-rollback")
	require.NoError(t, os.Mkdir(rollback, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(rollback, "saved"), []byte("old"), 0600))
	s.Start()
	require.Error(t, replace(t.TempDir()))
	data, err := os.ReadFile(filepath.Join(rollback, "saved"))
	require.NoError(t, err)
	require.Equal(t, "old", string(data))
}
