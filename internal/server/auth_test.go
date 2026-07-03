package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phin-tech/whisk/internal/server"
)

func TestControlAuthRequiresBearerExceptHealth(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /v1/compat", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /v1/ptys/{ptyID}/attach", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := server.RequireBearerAuth("secret", mux)

	assertAuthStatus(t, handler, http.MethodGet, "/v1/health", "", http.StatusOK)
	assertAuthStatus(t, handler, http.MethodGet, "/v1/compat", "", http.StatusUnauthorized)
	assertAuthStatus(t, handler, http.MethodGet, "/v1/compat", "Bearer wrong", http.StatusUnauthorized)
	assertAuthStatus(t, handler, http.MethodGet, "/v1/compat", "Basic secret", http.StatusUnauthorized)
	assertAuthStatus(t, handler, http.MethodGet, "/v1/compat", "Bearer secret", http.StatusNoContent)
	assertAuthStatus(t, handler, http.MethodGet, "/v1/ptys/pty_01/attach?from=0&access_token=secret", "", http.StatusNoContent)
}

func assertAuthStatus(t *testing.T, handler http.Handler, method string, target string, auth string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s auth %q status = %d, want %d, body %q", method, target, auth, rec.Code, want, rec.Body.String())
	}
}
