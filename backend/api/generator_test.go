package api

import (
	"net/http"
	"net/http/httptest"
	"sentinel/backend/config"
	"sentinel/backend/generator"
	"sentinel/backend/steam"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedGBESetupsLookup(t *testing.T) {
	configuration := &config.File{ManagedGBESetups: []config.ManagedGBESetup{
		{AppID: "620", Name: "Portal 2", Path: "/game"},
		{AppID: "480"},
	}}
	service := &generator.Service{Config: configuration}
	router := NewRouter(configuration, nil, nil, nil, service).Handler()
	request := httptest.NewRequest(http.MethodGet, "/decky-backend/gbe-setup/managed", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[{"appId":"620","name":"Portal 2"}]`, response.Body.String())
}

func TestSearchGamesRejectsInvalidQuery(t *testing.T) {
	router := NewRouter(&config.File{}, &steam.Service{}, nil, nil, &generator.Service{Config: &config.File{}}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/decky-backend/games/search?query=", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestConfigEndpointDoesNotExposeManagedInstallPath(t *testing.T) {
	configuration := &config.File{ManagedGBESetups: []config.ManagedGBESetup{{AppID: "620", Name: "Portal 2", Path: "/secret/game"}}}
	router := NewRouter(configuration, nil, nil, nil, &generator.Service{Config: configuration}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/decky-backend/config", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "/secret/game")
}
