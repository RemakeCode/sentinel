//go:build !decky

package generator

import "github.com/wailsapp/wails/v3/pkg/application"

const EventAchievementSetupUpdate = "sentinel::achievement-setup-update"

func (s *Service) emit(update Update) {
	if app := application.Get(); app != nil {
		app.Event.Emit(EventAchievementSetupUpdate, update)
	}
}
