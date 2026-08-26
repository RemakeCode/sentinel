//go:build !decky

package generator

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	return s.Start(ctx)
}

func (s *Service) SelectTargetDLL() (string, error) {
	app := application.Get()
	if app == nil {
		return "", errors.New("application is unavailable")
	}
	return app.Dialog.OpenFile().
		SetTitle("Select a Steam API DLL").
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AllowsOtherFileTypes(false).
		AddFilter("Steam API DLL", "*.dll").
		PromptForSingleSelection()
}
