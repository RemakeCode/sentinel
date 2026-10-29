//go:build decky

package watcher

func (s *Service) emitDataUpdated() {
	if s.Events != nil {
		s.Events.SendEvent("dataUpdated", struct{}{})
	}
}
