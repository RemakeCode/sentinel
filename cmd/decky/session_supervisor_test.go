package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSessionWatcher struct {
	mu            sync.Mutex
	startupErrors []error
	startCalls    int
	started       chan struct{}
}

func (m *mockSessionWatcher) Startup(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startCalls++
	var err error
	if len(m.startupErrors) > 0 {
		err = m.startupErrors[0]
		m.startupErrors = m.startupErrors[1:]
	}
	if err == nil && m.started != nil {
		select {
		case m.started <- struct{}{}:
		default:
		}
	}
	return err
}

func (m *mockSessionWatcher) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalls
}

func TestDeckSessionSupervisor_FreshBootStartsWatcherWhenSessionBecomesActive(t *testing.T) {
	active := false
	watcher := &mockSessionWatcher{}
	supervisor := newDeckSessionSupervisor(watcher, func() bool { return active })
	now := time.Date(2026, time.July, 26, 13, 15, 43, 0, time.UTC)

	assert.False(t, supervisor.reconcile(context.Background(), now))
	assert.Equal(t, 0, watcher.calls())

	active = true
	assert.True(t, supervisor.reconcile(context.Background(), now.Add(deckSessionPollInterval)))
	assert.True(t, supervisor.reconcile(context.Background(), now.Add(2*deckSessionPollInterval)))

	assert.Equal(t, 1, watcher.calls())
}

func TestDeckSessionSupervisor_DesktopSessionRemainsIdle(t *testing.T) {
	watcher := &mockSessionWatcher{}
	supervisor := newDeckSessionSupervisor(watcher, func() bool { return false })
	now := time.Now()

	for i := range 5 {
		assert.False(t, supervisor.reconcile(context.Background(), now.Add(time.Duration(i)*deckSessionPollInterval)))
	}

	assert.Equal(t, 0, watcher.calls())
}

func TestDeckSessionSupervisor_DoesNotRestartAfterStartup(t *testing.T) {
	active := true
	watcher := &mockSessionWatcher{}
	supervisor := newDeckSessionSupervisor(watcher, func() bool { return active })
	now := time.Now()

	assert.True(t, supervisor.reconcile(context.Background(), now))
	active = false
	assert.True(t, supervisor.reconcile(context.Background(), now.Add(deckSessionPollInterval)))
	assert.True(t, supervisor.reconcile(context.Background(), now.Add(2*deckSessionPollInterval)))
	active = true
	assert.True(t, supervisor.reconcile(context.Background(), now.Add(3*deckSessionPollInterval)))

	assert.Equal(t, 1, watcher.calls())
}

func TestDeckSessionSupervisor_RetriesWatcherStartupWithCappedBackoff(t *testing.T) {
	watcher := &mockSessionWatcher{
		startupErrors: []error{errors.New("first failure"), errors.New("second failure"), nil},
	}
	supervisor := newDeckSessionSupervisor(watcher, func() bool { return true })
	supervisor.retryInitialDelay = 2 * time.Second
	supervisor.retryMaxDelay = 4 * time.Second
	supervisor.retryDelay = supervisor.retryInitialDelay
	now := time.Now()

	assert.False(t, supervisor.reconcile(context.Background(), now))
	assert.False(t, supervisor.reconcile(context.Background(), now.Add(time.Second)))
	assert.Equal(t, 1, watcher.calls())

	assert.False(t, supervisor.reconcile(context.Background(), now.Add(2*time.Second)))
	assert.False(t, supervisor.reconcile(context.Background(), now.Add(5*time.Second)))
	assert.Equal(t, 2, watcher.calls())

	assert.True(t, supervisor.reconcile(context.Background(), now.Add(6*time.Second)))
	assert.Equal(t, 3, watcher.calls())
}

func TestDeckSessionSupervisor_StartupDoesNotStopWatcher(t *testing.T) {
	watcher := &mockSessionWatcher{
		started: make(chan struct{}, 1),
	}
	supervisor := newDeckSessionSupervisor(watcher, func() bool { return true })
	supervisor.pollInterval = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})

	go func() {
		supervisor.Run(ctx)
		close(done)
	}()

	select {
	case <-watcher.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for watcher startup")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for startup worker to finish")
	}

	assert.Equal(t, 1, watcher.calls())
}

func TestDeckSessionSupervisor_StopsWaitingAfterStartupGracePeriod(t *testing.T) {
	watcher := &mockSessionWatcher{}
	supervisor := newDeckSessionSupervisor(watcher, func() bool { return false })
	supervisor.pollInterval = time.Millisecond
	supervisor.startupTimeout = 10 * time.Millisecond
	done := make(chan struct{})

	go func() {
		supervisor.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for startup grace period")
	}

	assert.Equal(t, 0, watcher.calls())
}

func TestStartDecky_StartsSupervisorAfterServices(t *testing.T) {
	var order []string
	serverErr := errors.New("server stopped")

	err := startDecky(
		func() error {
			order = append(order, "services")
			return nil
		},
		func() {
			order = append(order, "supervisor")
		},
		func() error {
			order = append(order, "server")
			return serverErr
		},
	)

	require.ErrorIs(t, err, serverErr)
	assert.Equal(t, []string{"services", "supervisor", "server"}, order)
}

func TestStartDecky_DoesNotStartSupervisorWhenServicesFail(t *testing.T) {
	serviceErr := errors.New("services failed")
	supervisorStarted := false
	serverStarted := false

	err := startDecky(
		func() error { return serviceErr },
		func() { supervisorStarted = true },
		func() error {
			serverStarted = true
			return nil
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, serviceErr)
	assert.False(t, supervisorStarted)
	assert.False(t, serverStarted)
}
