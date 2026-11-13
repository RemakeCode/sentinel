package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"sentinel/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
	wup "github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

type Service struct {
	ctx context.Context
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx

	app := application.Get()

	manifest := "update.json"
	if backend.IsFlatpak() {
		manifest = "update-flatpak.json"
	}

	provider, err := endpoint.New(endpoint.Config{
		URL:     "https://github.com/RemakeCode/sentinel/releases/latest/download/" + manifest,
		Channel: "stable",
	})
	if err != nil {
		slog.Error("Failed to configure update endpoint", "error", err)
		return nil
	}

	err = app.Updater.Init(wup.Config{
		CurrentVersion: backend.Version,
		Providers:      []wup.Provider{provider},
		Platform:       runtime.GOOS,
		Arch:           runtime.GOARCH,
		Window:         wup.WindowNone,
	})
	if err != nil {
		slog.Error("Failed to initialize desktop updater", "error", err)
		return nil
	}

	return nil
}

func (s *Service) Check() error {
	_, err := application.Get().Updater.Check(s.ctx)
	if err != nil {
		slog.Warn("Unable to find update", "error", err)
	}

	return err
}

func (s *Service) State() wup.State {
	return application.Get().Updater.State()
}

func (s *Service) Download() error {
	return application.Get().Updater.DownloadAndInstall(s.ctx)
}

func (s *Service) Restart() error {
	return application.Get().Updater.Restart(s.ctx)
}

// Prepare handles package installations and updates. It must run before application initialization and after Wails helper handling.
func Prepare() error {
	packaged := "/usr/local/bin/sentinel"
	markerPattern := "/var/local/sentinel-install-*"
	if backend.IsFlatpak() {
		packaged = "/app/bin/sentinel"
		markerPattern = "/app/lib/sentinel/install-*"
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	directory := filepath.Join(backend.DataDir, "app")
	client := filepath.Join(directory, "sentinel")
	// Local development builds do not participate in package initialization.
	if executable != packaged && executable != client {
		return nil
	}

	markers, err := filepath.Glob(markerPattern)
	if err != nil {
		return err
	}
	if len(markers) == 0 && executable == client {
		// A retained client may still run after the system package is removed.
		return nil
	}
	if len(markers) != 1 {
		return errors.New("expected one Sentinel installation marker; reinstall the package")
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	marker := filepath.Join(directory, filepath.Base(markers[0]))
	_, markerErr := os.Stat(marker)
	if markerErr != nil && !errors.Is(markerErr, os.ErrNotExist) {
		return markerErr
	}

	replaced := errors.Is(markerErr, os.ErrNotExist)
	if replaced {
		if err := copyExecutable(packaged, client); err != nil {
			return fmt.Errorf("apply Sentinel package: %w", err)
		}

		if err := os.WriteFile(marker, nil, 0644); err != nil {
			return err
		}

		previous, err := filepath.Glob(filepath.Join(directory, filepath.Base(markerPattern)))
		if err != nil {
			return err
		}
		for _, path := range previous {
			if path != marker {
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
	}

	if executable == client && !replaced {
		return nil
	}

	return syscall.Exec(client, os.Args, os.Environ())
}

func copyExecutable(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	stage, err := os.CreateTemp(filepath.Dir(target), ".install-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	defer stage.Close()

	if _, err := io.Copy(stage, input); err != nil {
		return err
	}
	if err := stage.Chmod(0755); err != nil {
		return err
	}
	if err := stage.Sync(); err != nil {
		return err
	}
	if err := stage.Close(); err != nil {
		return err
	}

	return os.Rename(stage.Name(), target)
}
