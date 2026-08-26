package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sentinel/backend/config"
	"sentinel/backend/generator"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInspectGBEBackupsRejectsUnsupportedPath(t *testing.T) {
	service := &generator.Service{Config: &config.File{}}
	router := NewRouter(&config.File{}, nil, nil, nil, service).Handler()
	request := httptest.NewRequest(http.MethodPost, "/decky-backend/gbe-setup/preflight", bytes.NewBufferString(`{"appId":"620","dllPath":"/missing/steam_api64.dll"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestInspectGBEBackupsReturnsOnlyBackupStatus(t *testing.T) {
	directory := t.TempDir()
	dllPath := filepath.Join(directory, "steam_api64.dll")
	assert.NoError(t, os.WriteFile(dllPath, []byte("dll"), 0644))
	assert.NoError(t, os.WriteFile(dllPath+".sentinel.bak", []byte("backup"), 0644))

	service := &generator.Service{Config: &config.File{}}
	router := NewRouter(&config.File{}, nil, nil, nil, service).Handler()
	request := httptest.NewRequest(http.MethodPost, "/decky-backend/gbe-setup/preflight", bytes.NewBufferString(`{"appId":"620","dllPath":"`+dllPath+`"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"hasDllBackup":true,"hasSettingsBackup":false}`, response.Body.String())
}

func TestManagedGBESetupAppIDsLookup(t *testing.T) {
	configuration := &config.File{ManagedGBESetups: []config.ManagedGBESetup{
		{AppID: "620", Path: "/game"},
		{AppID: "480"},
	}}
	service := &generator.Service{Config: configuration}
	router := NewRouter(configuration, nil, nil, nil, service).Handler()
	request := httptest.NewRequest(http.MethodGet, "/decky-backend/gbe-setup/managed", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `["620"]`, response.Body.String())
}
