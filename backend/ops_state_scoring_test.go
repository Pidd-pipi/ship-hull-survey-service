package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpsQueuedCanCancel(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records/hull-1001/transition", bytes.NewBufferString(`{"expected":1,"status":"closed"}`)))
	if r.Code != http.StatusOK {
		t.Fatalf("cancel queued got %d, want 200 (body=%s)", r.Code, r.Body.String())
	}
}

func TestOpsPausedCanResume(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records/hull-1003/transition", bytes.NewBufferString(`{"expected":1,"status":"active"}`)))
	if r.Code != http.StatusOK {
		t.Fatalf("resume got %d, want 200 (body=%s)", r.Code, r.Body.String())
	}
}

func TestOpsClosedCannotReopen(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records/hull-1002/transition", bytes.NewBufferString(`{"expected":1,"status":"closed"}`)))
	if r.Code != http.StatusOK {
		t.Fatalf("close got %d body=%s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records/hull-1002/transition", bytes.NewBufferString(`{"expected":2,"status":"active"}`)))
	if r.Code != http.StatusBadRequest {
		t.Fatalf("reopen got %d, want 400 (body=%s)", r.Code, r.Body.String())
	}
}

func TestOpsSearchActiveExcludesPaused(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/ops/records?status=active", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("got %d", r.Code)
	}
	var page OpsPage
	if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("active total = %d, want 1 (only hull-1002)", page.Total)
	}
	for _, item := range page.Items {
		if item.Status == OpsStatusPaused {
			t.Fatalf("paused record leaked into active filter: %s", item.ID)
		}
	}
}

func TestOpsCanMoveClosedToActiveFalse(t *testing.T) {
	m := newOpsStateMachine()
	if m.CanMove(OpsStatusClosed, OpsStatusActive) {
		t.Fatal("closed -> active must be rejected by the transition table")
	}
}

func TestOpsCanMoveQueuedToClosedTrue(t *testing.T) {
	m := newOpsStateMachine()
	if !m.CanMove(OpsStatusQueued, OpsStatusClosed) {
		t.Fatal("queued -> closed must be allowed")
	}
}
