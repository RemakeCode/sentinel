package api

import (
	"net/http"
	"net/http/httptest"
	"sentinel/backend/config"
	"sentinel/backend/generator"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
