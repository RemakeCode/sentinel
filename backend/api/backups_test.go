package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sentinel/backend/backups"
	"sentinel/backend/config"

	"github.com/stretchr/testify/require"
)

func TestBackupRoutesRejectInvalidRequests(t *testing.T) {
	router := NewRouter(&config.File{}, nil, nil, nil, nil, nil, nil).Handler()
	for _, test := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/backups/restore", `{}`},
		{http.MethodPost, "/backups/restore", `not json`},
		{http.MethodPut, "/config/backup-directory", `{"path":" "}`},
		{http.MethodPut, "/config/backup-directory", `not json`},
	} {
		t.Run(test.method+test.path+test.body, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, "/decky-backend"+test.path, strings.NewReader(test.body)))
			require.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
}

func TestBackupConflictResponse(t *testing.T) {
	response := httptest.NewRecorder()
	handler := Wrap(func(http.ResponseWriter, *http.Request) error {
		return backupRequestError(backups.ErrOperationRunning)
	})
	handler(response, httptest.NewRequest(http.MethodPost, "/decky-backend/backups", nil))
	require.Equal(t, http.StatusConflict, response.Code)
	require.Contains(t, response.Body.String(), backups.ErrOperationRunning.Error())
}
