package main

import (
	"context"
	"log/slog"
	"time"
)

const (
	deckSessionPollInterval   = 2 * time.Second
	deckSessionStartupTimeout = 60 * time.Second
	watcherRetryInitialDelay  = 2 * time.Second
	watcherRetryMaxDelay      = 30 * time.Second
)

type deckSessionWatcher interface {
	Startup(context.Context) error
}

type deckSessionSupervisor struct {
	watcher           deckSessionWatcher
	isSessionActive   func() bool
	pollInterval      time.Duration
	startupTimeout    time.Duration
	retryInitialDelay time.Duration
	retryMaxDelay     time.Duration

	watcherStarted bool
	waitingLogged  bool
	retryDelay     time.Duration
	nextRetry      time.Time
}

func newDeckSessionSupervisor(watcher deckSessionWatcher, isSessionActive func() bool) *deckSessionSupervisor {
	return &deckSessionSupervisor{
		watcher:           watcher,
		isSessionActive:   isSessionActive,
		pollInterval:      deckSessionPollInterval,
		startupTimeout:    deckSessionStartupTimeout,
		retryInitialDelay: watcherRetryInitialDelay,
		retryMaxDelay:     watcherRetryMaxDelay,
		retryDelay:        watcherRetryInitialDelay,
	}
}

// Run handles the fresh-login race where Decky's system service can start before
// Gamescope and Steam's -gamepadui mode. A one-shot session check would then see
// a desktop session and permanently skip starting the watcher, so this waits
// briefly for Game Mode readiness and exits after the first successful start.
func (s *deckSessionSupervisor) Run(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	if s.reconcile(ctx, time.Now()) {
		return
	}

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	deadline := time.NewTimer(s.startupTimeout)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			slog.Info("Deck session was not ready during startup grace period; watcher remains stopped", "timeout", s.startupTimeout)
			return
		case now := <-ticker.C:
			if s.reconcile(ctx, now) {
				return
			}
		}
	}
}

func (s *deckSessionSupervisor) reconcile(ctx context.Context, now time.Time) bool {
	if s.watcherStarted {
		return true
	}

	if !s.isSessionActive() {
		if !s.waitingLogged {
			slog.Info("Waiting for active Deck session before starting watcher")
			s.waitingLogged = true
		}
		return false
	}

	if !s.nextRetry.IsZero() && now.Before(s.nextRetry) {
		return false
	}

	if err := s.watcher.Startup(ctx); err != nil {
		delay := s.retryDelay
		if delay <= 0 {
			delay = s.retryInitialDelay
		}
		s.nextRetry = now.Add(delay)
		s.retryDelay = min(delay*2, s.retryMaxDelay)
		slog.Warn("Watcher startup failed during Deck session readiness check; will retry", "error", err, "retryIn", delay)
		return false
	}

	s.watcherStarted = true
	s.resetRetry()
	slog.Info("Deck session watcher startup complete")
	return true
}

func (s *deckSessionSupervisor) resetRetry() {
	s.retryDelay = s.retryInitialDelay
	s.nextRetry = time.Time{}
}
