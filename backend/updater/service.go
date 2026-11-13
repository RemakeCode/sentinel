package updater

import (
	"context"
	"log/slog"

	wup "github.com/wailsapp/wails/v3/pkg/updater"
)

type Service struct {
	engine engine
	ctx    context.Context
}

type engine interface {
	State() wup.State
	Check(context.Context) (*wup.Release, error)
	DownloadAndInstall(context.Context) error
	Restart(context.Context) error
}

func (s *Service) Check() error {
	_, err := s.engine.Check(s.ctx)
	if err != nil {
		slog.Warn("Unable to find update", "error", err)
	}

	return err
}

func (s *Service) State() wup.State {
	return s.engine.State()
}

func (s *Service) Download() error {
	return s.engine.DownloadAndInstall(s.ctx)
}

func (s *Service) Restart() error {
	return s.engine.Restart(s.ctx)
}
