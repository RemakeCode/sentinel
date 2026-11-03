package backups

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"sentinel/backend"
	"sentinel/backend/config"
)

var cacheFolders = []string{"data", "games", "icon"}

var ErrOperationRunning = errors.New("a backup or restore is already running")

const maxExtractedBytes int64 = 20 << 30

type Status struct {
	ID        string `json:"id"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

type Snapshot struct {
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp"`
}

type Service struct {
	Config  *config.File
	Events  EventSink
	mu      sync.Mutex
	running bool
}

type EventSink interface {
	SendEvent(string, any)
}

// Start reports recovery data from an interrupted restore without deleting it.
//
//wails:internal
func (s *Service) Start() {
	rollback := filepath.Join(backend.DataDir, ".snapshot-rollback")
	if _, err := os.Stat(rollback); err == nil {
		slog.Error("interrupted snapshot restore; recovery data retained", "path", rollback)
	} else if !errors.Is(err, os.ErrNotExist) {
		slog.Error("cannot inspect snapshot recovery data", "path", rollback, "error", err)
	}
}

func (s *Service) Backup() error {
	return s.run("backup", func() error { return s.backup() })
}

func (s *Service) Restore(archive string) error {
	return s.run("restore", func() error { return s.restore(archive) })
}

func (s *Service) List() ([]Snapshot, error) {
	if s.Config == nil {
		return nil, errors.New("backup configuration unavailable")
	}

	entries, err := os.ReadDir(s.Config.BackupDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return []Snapshot{}, nil
	}
	if err != nil {
		return nil, err
	}

	var result []Snapshot
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "sentinel-") || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}

		stamp := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "sentinel-"), ".zip")
		stamp = strings.SplitN(stamp, "-", 2)[0]
		value, err := strconv.ParseInt(stamp, 10, 64)
		if err == nil {
			result = append(result, Snapshot{Name: entry.Name(), Timestamp: value})
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Timestamp > result[j].Timestamp })
	return result, nil
}

func (s *Service) Delete(name string) error {
	if s.Config == nil {
		return errors.New("backup configuration unavailable")
	}
	if name != filepath.Base(name) || name == "." || !strings.HasPrefix(name, "sentinel-") || !strings.HasSuffix(name, ".zip") {
		return errors.New("invalid snapshot name")
	}

	return os.Remove(filepath.Join(s.Config.BackupDirectory, name))
}

func (s *Service) run(operation string, work func() error) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrOperationRunning
	}

	s.running = true
	s.mu.Unlock()

	defer func() { s.mu.Lock(); s.running = false; s.mu.Unlock() }()

	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	s.emit(Status{ID: id, Operation: operation, Status: "running"})

	err := work()
	status := Status{ID: id, Operation: operation, Status: "succeeded"}
	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
	}

	s.emit(status)

	return err
}

func (s *Service) backup() (err error) {
	if s.Config == nil || s.Config.BackupDirectory == "" {
		return errors.New("backup directory is not configured")
	}

	if err = os.MkdirAll(s.Config.BackupDirectory, 0755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	var file *os.File
	for i := 0; ; i++ {
		name := fmt.Sprintf("sentinel-%d", time.Now().Unix())
		if i > 0 {
			name += fmt.Sprintf("-%d", i)
		}

		file, err = os.OpenFile(filepath.Join(s.Config.BackupDirectory, name+".zip"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if !errors.Is(err, os.ErrExist) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	defer func() {
		if err != nil {
			os.Remove(file.Name())
		}
	}()

	z := zip.NewWriter(file)
	for _, folder := range cacheFolders {
		root := filepath.Join(backend.DataDir, folder)
		if _, err = z.Create(folder + "/"); err != nil {
			break
		}

		_, statErr := os.Stat(root)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			err = statErr
			break
		}

		err = filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if p == root {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 || (!d.IsDir() && !d.Type().IsRegular()) {
				return fmt.Errorf("unsupported cache entry: %s", p)
			}

			rel, e := filepath.Rel(backend.DataDir, p)
			if e != nil {
				return e
			}

			name := filepath.ToSlash(rel)
			if d.IsDir() {
				name += "/"
			}

			w, e := z.Create(name)
			if e != nil || d.IsDir() {
				return e
			}

			in, e := os.Open(p)
			if e != nil {
				return e
			}
			defer in.Close()

			_, e = io.Copy(w, in)
			return e
		})

		if err != nil {
			break
		}
	}

	if e := z.Close(); err == nil {
		err = e
	}
	if e := file.Close(); err == nil {
		err = e
	}
	if err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	return nil
}

func (s *Service) restore(archive string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("open snapshot: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(backend.DataDir, 0755); err != nil {
		return err
	}

	stage, err := os.MkdirTemp(backend.DataDir, ".snapshot-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	seen := map[string]bool{}
	var total int64
	for _, f := range reader.File {
		name := f.Name
		if strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.Clean(name) != strings.TrimSuffix(name, "/") || strings.Contains(name, "\x00") {
			return fmt.Errorf("unsafe archive path: %q", name)
		}

		parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
		if len(parts) == 0 || (parts[0] != "data" && parts[0] != "games" && parts[0] != "icon") {
			return fmt.Errorf("unexpected archive entry: %s", name)
		}
		if len(parts) == 1 {
			if !f.FileInfo().IsDir() {
				return fmt.Errorf("expected directory: %s", name)
			}
			seen[parts[0]] = true
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return fmt.Errorf("unsupported archive entry: %s", name)
		}
		if f.UncompressedSize64 > uint64(maxExtractedBytes) || total > maxExtractedBytes-int64(f.UncompressedSize64) {
			return errors.New("snapshot exceeds extraction limit")
		}

		total += int64(f.UncompressedSize64)
		dest := filepath.Join(stage, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		in, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			in.Close()
			return err
		}

		_, copyErr := io.Copy(out, io.LimitReader(in, int64(f.UncompressedSize64)+1))
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	for _, folder := range cacheFolders {
		if !seen[folder] {
			return fmt.Errorf("snapshot missing %s directory", folder)
		}
	}

	return replace(stage)
}

func replace(stage string) error {
	rollback := filepath.Join(backend.DataDir, ".snapshot-rollback")
	if err := os.Mkdir(rollback, 0700); err != nil {
		return fmt.Errorf("cannot start restore; check retained recovery data at %s: %w", rollback, err)
	}

	for _, folder := range cacheFolders {
		target := filepath.Join(backend.DataDir, folder)
		if err := os.Rename(target, filepath.Join(rollback, folder)); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return rollbackFailure(rollback, err)
			}
			if err := os.WriteFile(filepath.Join(rollback, folder+".absent"), nil, 0600); err != nil {
				return rollbackFailure(rollback, err)
			}
		}

		if err := os.Rename(filepath.Join(stage, folder), target); err != nil {
			return rollbackFailure(rollback, err)
		}
	}

	if err := os.RemoveAll(rollback); err != nil {
		return fmt.Errorf("remove previous cache at %s: %w", rollback, err)
	}

	return nil
}

func rollbackFailure(rollback string, cause error) error {
	var failures []error
	for _, folder := range cacheFolders {
		old := filepath.Join(rollback, folder)
		if _, err := os.Lstat(old); errors.Is(err, os.ErrNotExist) {
			marker := old + ".absent"
			if _, markerErr := os.Stat(marker); markerErr == nil {
				if removeErr := os.RemoveAll(filepath.Join(backend.DataDir, folder)); removeErr != nil {
					failures = append(failures, removeErr)
					continue
				}
				if removeErr := os.Remove(marker); removeErr != nil {
					failures = append(failures, removeErr)
				}
			} else if !errors.Is(markerErr, os.ErrNotExist) {
				failures = append(failures, markerErr)
			}
			continue
		} else if err != nil {
			failures = append(failures, err)
			continue
		}

		target := filepath.Join(backend.DataDir, folder)
		if err := os.RemoveAll(target); err != nil {
			failures = append(failures, err)
			continue
		}

		if err := os.Rename(old, target); err != nil {
			failures = append(failures, err)
		}
	}

	if len(failures) != 0 {
		return fmt.Errorf("restore failed: %w; recovery data retained at %s: %v", cause, rollback, errors.Join(failures...))
	}
	if err := os.Remove(rollback); err != nil {
		return fmt.Errorf("restore failed: %w; recovery cleanup: %v", cause, err)
	}

	return fmt.Errorf("restore failed: %w", cause)
}
