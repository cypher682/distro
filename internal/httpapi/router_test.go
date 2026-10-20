package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cypher682/distro/internal/core"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func testRouter(pinger Pinger) http.Handler {
	return NewRouter(pinger, core.NewService(core.NewMemoryStore()))
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	testRouter(fakePinger{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "{\"service\":\"distro\",\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestHealthDBUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health/db", nil)
	rec := httptest.NewRecorder()

	testRouter(fakePinger{err: errors.New("down")}).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHealthDBOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health/db", nil)
	rec := httptest.NewRecorder()

	testRouter(fakePinger{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
