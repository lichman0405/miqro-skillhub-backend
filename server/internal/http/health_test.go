package http_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpx "miqro-skillhub/server/internal/http"
)

func TestHealthz(t *testing.T) {
	h := &httpx.HealthHandler{}
	mux := http.NewServeMux()
	h.RegisterHealthRoutes(mux)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReadyz_Default(t *testing.T) {
	h := &httpx.HealthHandler{}
	mux := http.NewServeMux()
	h.RegisterHealthRoutes(mux)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReadyz_NotReady(t *testing.T) {
	h := &httpx.HealthHandler{
		Ready: func() error {
			return errors.New("db not connected")
		},
	}
	mux := http.NewServeMux()
	h.RegisterHealthRoutes(mux)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestReadyz_Ready(t *testing.T) {
	h := &httpx.HealthHandler{
		Ready: func() error {
			return nil
		},
	}
	mux := http.NewServeMux()
	h.RegisterHealthRoutes(mux)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
