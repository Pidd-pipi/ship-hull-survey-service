package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpsDuplicateCreateConflict(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	body := `{"id":"hull-1001","subject":"Port bow coating inspection","owner":"surveyor-lin","status":"queued","priority":"high","labels":{"site":"MV Aurora","operator":"surveyor-lin","evidence":"photo-1"}}`
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records", bytes.NewBufferString(body)))
	if r.Code != http.StatusConflict {
		t.Fatalf("duplicate create got %d, want 409 (body=%s)", r.Code, r.Body.String())
	}
}

func TestOpsCreatePolicyBadRequest(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	body := `{"id":"hull-2001","subject":"Keel plate scan","status":"queued","priority":"high","labels":{"site":"MV Orion","operator":"surveyor-wu","evidence":"scan-1"}}`
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records", bytes.NewBufferString(body)))
	if r.Code != http.StatusBadRequest {
		t.Fatalf("policy create got %d, want 400 (body=%s)", r.Code, r.Body.String())
	}
}

func TestOpsTransitionInvalidStatus(t *testing.T) {
	h := newOpsHandler(newOpsService(opsSeed()))
	r := httptest.NewRecorder()
	body := `{"expected":1,"status":"bogus"}`
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records/hull-1001/transition", bytes.NewBufferString(body)))
	if r.Code != http.StatusBadRequest {
		t.Fatalf("invalid transition got %d, want 400 (body=%s)", r.Code, r.Body.String())
	}
}
