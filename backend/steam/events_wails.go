//go:build !decky

package steam

import (
	"sentinel/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *Service) emitFetchStatus(current uint32, total uint32) {
	app := application.Get()
	if app != nil {
		app.Event.Emit(backend.EventFetchStatus, backend.FetchStatusEvt{Current: current, Total: total})
	}
}
