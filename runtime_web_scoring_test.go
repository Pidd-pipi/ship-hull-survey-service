package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDsUniquePerRequest(t *testing.T) {
	h := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r1 := httptest.NewRecorder()
	h.ServeHTTP(r1, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	r2 := httptest.NewRecorder()
	h.ServeHTTP(r2, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	id1 := r1.Header().Get("X-Request-ID")
	id2 := r2.Header().Get("X-Request-ID")
	if id1 == "" || id2 == "" {
		t.Fatalf("missing request ids: %q %q", id1, id2)
	}
	if id1 == id2 {
		t.Fatalf("request ids reused: %q", id1)
	}
}

func TestStaticUnknownPathNotFound(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/some/unknown/path", nil))
	if r.Code != http.StatusNotFound {
		t.Fatalf("unknown path got %d, want 404", r.Code)
	}
}

func TestRecoveryMiddlewareCatchesPanic(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	h := recoveryMiddleware(inner)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if r.Code != http.StatusInternalServerError {
		t.Fatalf("panic handler got %d, want 500", r.Code)
	}
}

func TestStaticRootServesIndex(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("root got %d", r.Code)
	}
	if len(r.Body.String()) < 50 {
		t.Fatalf("root body too short")
	}
}

func TestStaticNotLongCached(t *testing.T) {
	h := staticHandler()
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := r.Header().Get("Cache-Control"); got == "public, max-age=86400" {
		t.Fatal("static page cached for a full day")
	}
}
