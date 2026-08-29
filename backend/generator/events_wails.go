//go:build !decky

package generator

import "github.com/wailsapp/wails/v3/pkg/application"

const (
	EventAchievementSetupSelected = "sentinel::achievement-setup-selected"
	EventAchievementSetupUpdate   = "sentinel::achievement-setup-update"
)

type AchievementSetupAction string

const (
	AchievementSetupActionSetup AchievementSetupAction = "setup"
	AchievementSetupActionUndo  AchievementSetupAction = "undo"
)

type AchievementSetupSelection struct {
	Action   AchievementSetupAction `json:"action"`
	AppID    string                 `json:"appId"`
	GameName string                 `json:"gameName"`
}

func (s *Service) emit(update Update) {
	if app := application.Get(); app != nil {
		app.Event.Emit(EventAchievementSetupUpdate, update)
	}
}
