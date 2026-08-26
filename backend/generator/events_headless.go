//go:build decky

package generator

func (s *Service) emit(update Update) {
	if s.Events != nil {
		s.Events.SendEvent("gbeSetup", update)
	}
}
