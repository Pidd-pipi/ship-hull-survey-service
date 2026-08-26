package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEvidenceInvalidKindRejected(t *testing.T) {
	findings := newFindingStore()
	h := newEvidenceHandler(findings, newEvidenceStore())
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/evidence/sf-301", bytes.NewBufferString(`{"kind":"bogus","note":"x"}`)))
	if r.Code != http.StatusBadRequest {
		t.Fatalf("invalid kind got %d, want 400 (body=%s)", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/evidence/sf-301", nil))
	var refs []EvidenceRef
	if err := json.Unmarshal(r.Body.Bytes(), &refs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("invalid evidence was stored: %d refs", len(refs))
	}
}

func TestEvidenceListBounded(t *testing.T) {
	store := newEvidenceStore()
	for i := 0; i < maxEvidencePerFinding+5; i++ {
		if _, err := store.Add("sf-301", "photo", "note"); err != nil && i < maxEvidencePerFinding {
			t.Fatalf("add %d failed: %v", i, err)
		}
	}
	if got := len(store.For("sf-301")); got != maxEvidencePerFinding {
		t.Fatalf("evidence list = %d, want %d (bounded)", got, maxEvidencePerFinding)
	}
}

func TestEvidenceForReturnsCopy(t *testing.T) {
	store := newEvidenceStore()
	if _, err := store.Add("sf-301", "photo", "original note"); err != nil {
		t.Fatalf("add: %v", err)
	}
	refs := store.For("sf-301")
	refs[0].Note = "tampered"
	again := store.For("sf-301")
	if again[0].Note != "original note" {
		t.Fatalf("evidence list aliases internal storage: %q", again[0].Note)
	}
}

func TestValidateEvidenceKindScan(t *testing.T) {
	if err := validateEvidenceKind("scan"); err != nil {
		t.Fatalf("scan is a legal evidence kind: %v", err)
	}
}
