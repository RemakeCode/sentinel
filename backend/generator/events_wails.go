//go:build !decky

package generator

import "github.com/wailsapp/wails/v3/pkg/application"

const (
	EventGBESetup        = "sentinel::gbe-setup"
	EventGBESetupRequest = "sentinel::gbe-setup-requested"
	EventGBEUndoRequest  = "sentinel::gbe-undo-requested"
)

type SetupDialogRequest struct {
	AppID    string `json:"appId"`
	GameName string `json:"gameName"`
}

type UndoDialogRequest struct {
	AppID    string `json:"appId"`
	GameName string `json:"gameName"`
}

func (s *Service) emit(update Update) {
	if app := application.Get(); app != nil {
		app.Event.Emit(EventGBESetup, update)
	}
}
