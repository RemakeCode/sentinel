package updater

import (
	"context"
	"fmt"
	"log/slog"

	"sentinel/backend"

	wup "github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
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

func installation() string {
	if backend.IsFlatpak() {
		return "flatpak"
	}

	return "native"
}

func matchArtifact(kind string) github.AssetMatcher {
	return func(request wup.CheckRequest, assets []github.ReleaseAsset) int {
		prefix := "sentinel"
		if kind == "flatpak" {
			prefix += "-flatpak"
		}
		name := fmt.Sprintf("%s-%s-%s.tar.gz", prefix, request.Platform, request.Arch)
		for index, asset := range assets {
			if asset.Name == name {
				return index
			}
		}
		return -1
	}
}
