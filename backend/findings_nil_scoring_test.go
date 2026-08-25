package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFindingsEmptyListIsArray(t *testing.T) {
	store := newFindingStore()
	h := newRouter(store)
	// Remove every finding so the list is empty.
	for _, id := range []string{"sf-301", "sf-302"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodDelete, "/api/findings/"+id, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("delete %s got %d", id, r.Code)
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/findings", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("list got %d", r.Code)
	}
	body := strings.TrimSpace(r.Body.String())
	if !strings.HasPrefix(body, "[") {
		t.Fatalf("empty list is not a JSON array: %s", body)
	}
}

func TestFindingMissingReturns404(t *testing.T) {
	h := newRouter(newFindingStore())
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/findings/nope-777", nil))
	if r.Code != http.StatusNotFound {
		t.Fatalf("missing get got %d, want 404 (body=%s)", r.Code, r.Body.String())
	}
}

func TestFindingEmptyListBodyNotNull(t *testing.T) {
	store := newFindingStore()
	h := newRouter(store)
	for _, id := range []string{"sf-301", "sf-302"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/api/findings/"+id, nil))
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/findings", nil))
	if bytes.Contains(r.Body.Bytes(), []byte("null")) && !bytes.Contains(r.Body.Bytes(), []byte("[]")) {
		t.Fatalf("empty list serialized as null: %s", r.Body.String())
	}
}

func TestFindingMissingGetReturnsError(t *testing.T) {
	store := newFindingStore()
	if _, err := store.get("nope-888"); err == nil {
		t.Fatal("missing finding returned no error")
	}
}

func TestFindingMissingDeleteReturns404(t *testing.T) {
	h := newRouter(newFindingStore())
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodDelete, "/api/findings/nope-888", nil))
	if r.Code != http.StatusNotFound {
		t.Fatalf("missing delete got %d, want 404", r.Code)
	}
}
