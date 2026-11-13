//go:build decky

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sentinel/backend"
	"sentinel/backend/backups"
	"sentinel/backend/config"
	"sentinel/backend/notifier"

	"github.com/stretchr/testify/require"
)

func TestBackupStatusThroughDeckySSE(t *testing.T) {
	previousDataDir := backend.DataDir
	backend.DataDir = t.TempDir()
	t.Cleanup(func() { backend.DataDir = previousDataDir })
	configuration := &config.File{BackupDirectory: t.TempDir()}
	notifications := &notifier.Service{Config: configuration}
	require.NoError(t, notifications.Start(context.Background()))
	defer notifications.ServiceShutdown()

	service := &backups.Service{Config: configuration, Events: notifications}
	router := NewRouter(configuration, nil, nil, notifications, nil, service, nil).Handler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	response := &synchronizedSSEWriter{header: make(http.Header)}
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/decky-backend/notifications", nil).WithContext(ctx))
		close(done)
	}()
	require.Eventually(t, func() bool { return notifications.ClientCount() == 1 }, time.Second, time.Millisecond)
	require.NoError(t, service.Backup())
	invalidArchive := filepath.Join(t.TempDir(), "invalid.zip")
	require.NoError(t, os.WriteFile(invalidArchive, []byte("invalid archive"), 0600))
	require.Error(t, service.Restore(invalidArchive))
	require.Eventually(t, func() bool { return strings.Contains(response.String(), `"status":"failed"`) }, time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not stop")
	}

	body := response.String()
	require.Equal(t, 4, strings.Count(body, `"messageType":"backupStatus"`))
	require.Contains(t, body, `"operation":"backup","status":"running"`)
	require.Contains(t, body, `"operation":"backup","status":"succeeded"`)
	require.Contains(t, body, `"operation":"restore","status":"running"`)
	require.Contains(t, body, `"operation":"restore","status":"failed","error":`)
}
