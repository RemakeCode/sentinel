package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sentinel/backend/config"
	"sentinel/backend/notifier"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type synchronizedSSEWriter struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
}

func (w *synchronizedSSEWriter) Header() http.Header { return w.header }
func (w *synchronizedSSEWriter) Write(value []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(value)
}
func (w *synchronizedSSEWriter) WriteHeader(_ int) {}
func (w *synchronizedSSEWriter) Flush()            {}
func (w *synchronizedSSEWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

func TestNotificationsSSEDeliversRealEnvelopeThroughHTTPHandler(t *testing.T) {
	service := &notifier.Service{Config: &config.File{}}
	require.NoError(t, service.Start(context.Background()))
	defer service.ServiceShutdown()

	router := NewRouter(nil, nil, nil, service, nil).Handler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	request := httptest.NewRequest("GET", "/decky-backend/notifications", nil).WithContext(ctx)
	response := &synchronizedSSEWriter{header: make(http.Header)}
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(response, request)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for service.ClientCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, 1, service.ClientCount())
	service.SendEvent("gbeSetup", map[string]string{"phase": "generating"})
	service.SendEvent("achievement", map[string]string{"Title": "Unlocked"})
	deadline = time.Now().Add(time.Second)
	for !strings.Contains(response.String(), `"messageType":"achievement"`) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not stop after request cancellation")
	}

	body := response.String()
	require.True(t, strings.Contains(body, `"messageType":"gbeSetup"`))
	require.True(t, strings.Contains(body, `"phase":"generating"`))
	require.True(t, strings.Contains(body, `"messageType":"achievement"`))
	require.True(t, strings.Contains(body, `"Title":"Unlocked"`))
}
