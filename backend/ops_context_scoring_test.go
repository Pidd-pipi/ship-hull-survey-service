package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpsContextParentCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	derived, release := opsContext(ctx, 3*time.Second)
	defer release()
	select {
	case <-derived.Done():
		// parent cancellation must propagate
	default:
		t.Fatal("opsContext dropped parent cancellation")
	}
}

func TestOpsTransitionCanceledContext(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records/hull-1001/transition", bytes.NewBufferString(`{"expected":1,"status":"active"}`)).WithContext(ctx)
	h.ServeHTTP(r, req)
	if r.Code == http.StatusOK {
		t.Fatalf("canceled transition committed: %d body=%s", r.Code, r.Body.String())
	}
	r2 := httptest.NewRecorder()
	h.ServeHTTP(r2, httptest.NewRequest(http.MethodGet, "/api/ops/records/hull-1001", nil))
	var record OpsRecord
	if err := json.Unmarshal(r2.Body.Bytes(), &record); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if record.Status != OpsStatusQueued {
		t.Fatalf("record status changed to %s after canceled transition", record.Status)
	}
}

func TestOpsCreateCanceledNotStored(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	body := `{"id":"hull-9001","subject":"Canceled create","owner":"surveyor-q","status":"queued","priority":"low","labels":{"site":"MV Q","operator":"surveyor-q","evidence":"e"}}`
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records", bytes.NewBufferString(body)).WithContext(ctx)
	h.ServeHTTP(r, req)
	if r.Code == http.StatusCreated {
		t.Fatalf("canceled create committed: %d body=%s", r.Code, r.Body.String())
	}
	r2 := httptest.NewRecorder()
	h.ServeHTTP(r2, httptest.NewRequest(http.MethodGet, "/api/ops/records/hull-9001", nil))
	if r2.Code != http.StatusNotFound {
		t.Fatalf("canceled record was stored, get=%d", r2.Code)
	}
}

func TestOpsSearchCanceledContext(t *testing.T) {
	svc := newOpsService(opsSeed())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Search(ctx, OpsQuery{}); err == nil {
		t.Fatal("search succeeded on canceled context")
	}
}

func TestOpsContextZeroTimeoutDefault(t *testing.T) {
	derived, release := opsContext(context.Background(), 0)
	defer release()
	select {
	case <-derived.Done():
		t.Fatal("opsContext with zero timeout expired immediately (default 5s expected)")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestOpsGetCanceledContext(t *testing.T) {
	svc := newOpsService(opsSeed())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Get(ctx, "hull-1001"); err == nil {
		t.Fatal("get succeeded on canceled context")
	}
}
