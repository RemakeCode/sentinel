package generator

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeTools struct{ prepared *PreparedTools }

func (f fakeTools) PrepareTools(context.Context, InstallTarget, func(string)) (*PreparedTools, error) {
	return f.prepared, nil
}

type failingTools struct{ called bool }

func (f *failingTools) PrepareTools(context.Context, InstallTarget, func(string)) (*PreparedTools, error) {
	f.called = true
	return nil, errors.New("asset preparation stopped")
}

type approvedAuth struct{}

func (approvedAuth) Begin(context.Context) (AuthChallenge, error) {
	return AuthChallenge{ClientID: 1, RequestID: []byte("request"), ChallengeURL: "challenge", Interval: time.Millisecond}, nil
}
func (approvedAuth) Poll(context.Context, uint64, []byte) (AuthPoll, error) {
	return AuthPoll{AccountName: "account", RefreshToken: "secret"}, nil
}

type pendingAuth struct{}

func (pendingAuth) Begin(context.Context) (AuthChallenge, error) {
	return AuthChallenge{ClientID: 1, RequestID: []byte("request"), ChallengeURL: "challenge", Interval: time.Millisecond}, nil
}

func (pendingAuth) Poll(ctx context.Context, _ uint64, _ []byte) (AuthPoll, error) {
	<-ctx.Done()
	return AuthPoll{}, ctx.Err()
}

type capturedEvents struct{ updates []Update }

func (c *capturedEvents) SendEvent(_ string, payload any) {
	c.updates = append(c.updates, payload.(Update))
}

func TestSetupGBEInstallsOnlySuccessfulOpaqueGSEOutput(t *testing.T) {
	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	executable := filepath.Join(gseDir, "generate_emu_config")
	script := "#!/bin/sh\nset -eu\ntest -f refresh_tokens.json\ntest \"$#\" -eq 1\nmkdir -p \"_OUTPUT/$1/steam_settings\"\nprintf '[{},{}]' > \"_OUTPUT/$1/steam_settings/achievements.json\"\nprintf '[{}]' > \"_OUTPUT/$1/steam_settings/stats.json\"\nprintf opaque > \"_OUTPUT/$1/steam_settings/custom.bin\"\n"
	require.NoError(t, os.WriteFile(executable, []byte(script), 0755))
	workspace := filepath.Join(root, "workspace")
	require.NoError(t, os.Mkdir(workspace, 0700))
	stagedDLL := filepath.Join(workspace, "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("gbe"), 0644))
	targetDir := filepath.Join(root, "game")
	require.NoError(t, os.Mkdir(targetDir, 0755))
	targetDLL := filepath.Join(targetDir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))

	originalConfigPath := backend.ConfigPath
	backend.ConfigPath = filepath.Join(root, "config", "config.json")
	t.Cleanup(func() { backend.ConfigPath = originalConfigPath })
	events := &capturedEvents{}
	service := &Service{
		Config: &config.File{}, Auth: approvedAuth{}, Events: events,
		Tools: fakeTools{prepared: &PreparedTools{GeneratorExecutable: executable, ReplacementDLL: stagedDLL, WorkspaceDir: workspace}},
	}
	result, err := service.SetupGBE(SetupRequest{AppID: "620", GameName: "Portal 2", DLLPath: targetDLL})
	require.NoError(t, err)
	require.Equal(t, "620", result.AppID)
	require.FileExists(t, filepath.Join(targetDir, steamSettingsDirectoryName, "custom.bin"))
	require.FileExists(t, targetDLL+sentinelBackupSuffix)
	require.NoFileExists(t, filepath.Join(gseDir, gseTokenFilename))
	require.NoDirExists(t, filepath.Join(gseDir, gseOutputDirectoryName, "620"))
	require.NoDirExists(t, workspace)
	require.Equal(t, []ManagedGBESetupSummary{{AppID: "620", Name: "Portal 2"}}, service.ManagedGBESetups())
	for _, update := range events.updates {
		require.NotContains(t, update.Message, "secret")
	}
}

func TestFailedGSELeftoverOutputNeverInstalls(t *testing.T) {
	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	executable := filepath.Join(gseDir, "generate_emu_config")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\nmkdir -p \"_OUTPUT/$1/steam_settings\"\nprintf partial > \"_OUTPUT/$1/steam_settings/partial.txt\"\nexit 1\n"), 0755))
	workspace := filepath.Join(root, "workspace")
	require.NoError(t, os.Mkdir(workspace, 0700))
	stagedDLL := filepath.Join(workspace, "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("gbe"), 0644))
	targetDir := filepath.Join(root, "game")
	require.NoError(t, os.Mkdir(targetDir, 0755))
	targetDLL := filepath.Join(targetDir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))
	service := &Service{
		Config: &config.File{}, Auth: approvedAuth{},
		Tools: fakeTools{prepared: &PreparedTools{GeneratorExecutable: executable, ReplacementDLL: stagedDLL, WorkspaceDir: workspace}},
	}
	_, err := service.SetupGBE(SetupRequest{AppID: "620", GameName: "Portal 2", DLLPath: targetDLL})
	require.Error(t, err)
	data, readErr := os.ReadFile(targetDLL)
	require.NoError(t, readErr)
	require.Equal(t, "original", string(data))
	require.NoFileExists(t, targetDLL+sentinelBackupSuffix)
	require.NoDirExists(t, filepath.Join(gseDir, gseOutputDirectoryName, "620"))
	require.Empty(t, service.ManagedGBESetups())
}

func TestSuccessfulGSEWithoutOutputNeverInstalls(t *testing.T) {
	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	executable := filepath.Join(gseDir, "generate_emu_config")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0755))
	workspace := filepath.Join(root, "workspace")
	require.NoError(t, os.Mkdir(workspace, 0700))
	stagedDLL := filepath.Join(workspace, "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("gbe"), 0644))
	targetDirectory := filepath.Join(root, "game")
	require.NoError(t, os.Mkdir(targetDirectory, 0755))
	targetDLL := filepath.Join(targetDirectory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))

	service := &Service{
		Config: &config.File{}, Auth: approvedAuth{},
		Tools: fakeTools{prepared: &PreparedTools{GeneratorExecutable: executable, ReplacementDLL: stagedDLL, WorkspaceDir: workspace}},
	}
	_, err := service.SetupGBE(SetupRequest{AppID: "620", GameName: "Portal 2", DLLPath: targetDLL})
	require.ErrorContains(t, err, "without an available generated output folder")
	require.Equal(t, []byte("original"), mustReadFile(t, targetDLL))
	require.NoFileExists(t, targetDLL+sentinelBackupSuffix)
}

func TestExistingBackupsDoNotBlockPreparation(t *testing.T) {
	directory := t.TempDir()
	targetDLL := filepath.Join(directory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("current"), 0644))
	require.NoError(t, os.WriteFile(targetDLL+sentinelBackupSuffix, []byte("original"), 0644))
	tools := &failingTools{}
	service := &Service{Config: &config.File{}, Tools: tools}

	_, err := service.SetupGBE(SetupRequest{AppID: "620", GameName: "Portal 2", DLLPath: targetDLL})
	require.ErrorContains(t, err, "asset preparation stopped")
	require.True(t, tools.called)
	require.Equal(t, []byte("current"), mustReadFile(t, targetDLL))
	require.Equal(t, []byte("original"), mustReadFile(t, targetDLL+sentinelBackupSuffix))
}

func TestSetupGBELogsOperationFailure(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	directory := t.TempDir()
	targetDLL := filepath.Join(directory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("current"), 0644))
	service := &Service{Config: &config.File{}, Tools: &failingTools{}}

	_, err := service.SetupGBE(SetupRequest{AppID: "620", GameName: "Portal 2", DLLPath: targetDLL})

	require.ErrorContains(t, err, "asset preparation stopped")
	require.Contains(t, logs.String(), "level=ERROR")
	require.Contains(t, logs.String(), "msg=\"GBE setup failed\"")
	require.Contains(t, logs.String(), "app_id=620")
	require.Contains(t, logs.String(), "error=\"asset preparation stopped\"")
}

func TestSetupGBETimesOutQRApproval(t *testing.T) {
	originalTimeout := qrApprovalTimeout
	qrApprovalTimeout = time.Millisecond
	t.Cleanup(func() { qrApprovalTimeout = originalTimeout })

	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	workspace := filepath.Join(root, "workspace")
	targetDir := filepath.Join(root, "game")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	require.NoError(t, os.Mkdir(workspace, 0700))
	require.NoError(t, os.Mkdir(targetDir, 0755))
	targetDLL := filepath.Join(targetDir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))

	service := &Service{
		Config: &config.File{}, Auth: pendingAuth{},
		Tools: fakeTools{prepared: &PreparedTools{
			GeneratorExecutable: filepath.Join(gseDir, "generate_emu_config"),
			ReplacementDLL:      filepath.Join(workspace, "steam_api64.dll"),
			WorkspaceDir:        workspace,
		}},
	}

	_, err := service.SetupGBE(SetupRequest{AppID: "620", GameName: "Portal 2", DLLPath: targetDLL})
	require.ErrorIs(t, err, errQRApprovalTimedOut)
	require.FileExists(t, targetDLL)
	require.NoFileExists(t, targetDLL+sentinelBackupSuffix)
}

func TestUndoRemovesStaleManagedEntryWithoutChangingDirectory(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "keep.txt")
	require.NoError(t, os.WriteFile(marker, []byte("keep"), 0644))
	originalConfigPath := backend.ConfigPath
	backend.ConfigPath = filepath.Join(t.TempDir(), "config.json")
	t.Cleanup(func() { backend.ConfigPath = originalConfigPath })
	configuration := &config.File{ManagedGBESetups: []config.ManagedGBESetup{{AppID: "620", Name: "Portal 2", Path: root}}}
	service := &Service{Config: configuration}
	require.NoError(t, service.UndoGBESetup("620"))
	require.Empty(t, service.ManagedGBESetups())
	require.FileExists(t, marker)
}

func TestCleanupRemovesTransientDataAndRetainsCachedTool(t *testing.T) {
	originalGeneratorDir := backend.GeneratorDir
	backend.GeneratorDir = t.TempDir()
	t.Cleanup(func() { backend.GeneratorDir = originalGeneratorDir })
	gseDir := filepath.Join(backend.GeneratorDir, gseForkToolsAsset.cacheDirectoryName, gseForkToolsAsset.version, "generate_emu_config")
	require.NoError(t, os.MkdirAll(filepath.Join(gseDir, gseOutputDirectoryName, "620", steamSettingsDirectoryName), 0755))
	alternateOutput := filepath.Join(backend.GeneratorDir, gseForkToolsAsset.cacheDirectoryName, gseForkToolsAsset.version, "normal-output", "620")
	require.NoError(t, os.MkdirAll(filepath.Join(alternateOutput, steamSettingsDirectoryName), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gseDir, gseTokenFilename), []byte("secret"), 0600))
	tool := filepath.Join(gseDir, "generate_emu_config")
	require.NoError(t, os.WriteFile(tool, []byte("tool"), 0755))
	cleanupGeneratorTransientData()
	require.FileExists(t, tool)
	require.NoFileExists(t, filepath.Join(gseDir, gseTokenFilename))
	require.NoDirExists(t, filepath.Join(gseDir, gseOutputDirectoryName))
	require.DirExists(t, alternateOutput)
}

func TestCleanupRemovesOperationAndAssetTemporaryPaths(t *testing.T) {
	originalGeneratorDir := backend.GeneratorDir
	backend.GeneratorDir = t.TempDir()
	t.Cleanup(func() { backend.GeneratorDir = originalGeneratorDir })

	operations := filepath.Join(backend.GeneratorDir, "operations", "operation-1")
	require.NoError(t, os.MkdirAll(operations, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(operations, "steam_api64.dll"), []byte("dll"), 0644))
	temporaryDirectory := filepath.Join(backend.GeneratorDir, tempDirName)
	require.NoError(t, os.MkdirAll(filepath.Join(temporaryDirectory, "gse-extract-test"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(temporaryDirectory, ".download-test"), []byte("archive"), 0644))

	cleanupGeneratorTransientData()
	require.NoDirExists(t, filepath.Join(backend.GeneratorDir, "operations"))
	require.NoDirExists(t, temporaryDirectory)
}

func TestRunGSETerminatesWhenCancelled(t *testing.T) {
	directory := t.TempDir()
	startedPath := filepath.Join(directory, "started")
	t.Setenv("SENTINEL_GSE_TEST_STARTED_PATH", startedPath)
	executable, err := os.Executable()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := runGSEGenerator(ctx, executable, "-test.run=^TestGSEHelperProcess$")
		errCh <- err
	}()

	require.Eventually(t, func() bool {
		_, err := os.Stat(startedPath)
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)

	started := time.Now()
	cancel()
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("generator did not terminate after cancellation")
	}
	require.Less(t, time.Since(started), 2*time.Second)
	require.Equal(t, 5*time.Minute, generatorTimeout)
}

func TestGSEHelperProcess(t *testing.T) {
	startedPath := os.Getenv("SENTINEL_GSE_TEST_STARTED_PATH")
	if startedPath == "" {
		return
	}

	require.NoError(t, os.WriteFile(startedPath, []byte("started"), 0600))
	for {
		time.Sleep(time.Hour)
	}
}
