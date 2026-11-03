//go:build decky

package backups

func (s *Service) emit(status Status) {
	if s.Events != nil {
		s.Events.SendEvent("backupStatus", status)
	}
}
