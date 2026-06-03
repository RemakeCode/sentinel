package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouterAuthValidBearerToken(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/decky-backend/ready", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()

	router.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouterAuthValidQueryToken(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/decky-backend/ready?decky_auth_token=secret-token", nil)
	rec := httptest.NewRecorder()

	router.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouterAuthMissingToken(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/decky-backend/ready", nil)
	rec := httptest.NewRecorder()

	router.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.NotContains(t, rec.Body.String(), "secret-token")
}

func TestRouterAuthInvalidToken(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/decky-backend/ready", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()

	router.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.NotContains(t, rec.Body.String(), "secret-token")
}

func TestRouterAuthAllowsOptionsPreflight(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, "secret-token")
	req := httptest.NewRequest(http.MethodOptions, "/decky-backend/ready", nil)
	req.Header.Set("Origin", "http://localhost:1337")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()

	router.Handler().ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}
