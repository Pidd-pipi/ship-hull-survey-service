package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpsAPI(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		h := newOpsHandler(newOpsService(opsSeed()))
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/ops/records", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("got %d", r.Code)
		}
		var page OpsPage
		if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if page.Total != 3 {
			t.Fatalf("total = %d", page.Total)
		}
	})

	t.Run("create", func(t *testing.T) {
		h := newOpsHandler(newOpsService(opsSeed()))
		r := httptest.NewRecorder()
		body := `{"id":"hull-2001","subject":"Keel plate thickness scan","owner":"surveyor-wu","status":"queued","priority":"high","labels":{"site":"MV Orion","operator":"surveyor-wu","evidence":"scan-1"}}`
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records", bytes.NewBufferString(body)))
		if r.Code != http.StatusCreated {
			t.Fatalf("got %d body=%s", r.Code, r.Body.String())
		}
	})

	t.Run("snapshot", func(t *testing.T) {
		h := newOpsHandler(newOpsService(opsSeed()))
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/ops/snapshot", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("got %d", r.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newOpsHandler(newOpsService(opsSeed()))
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/ops/records", bytes.NewBufferString("{not-json")))
		if r.Code != http.StatusBadRequest {
			t.Fatalf("got %d", r.Code)
		}
	})
}

func TestEvidenceAPI(t *testing.T) {
	t.Run("add and list", func(t *testing.T) {
		findings := newFindingStore()
		h := newEvidenceHandler(findings, newEvidenceStore())
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/evidence/sf-301", bytes.NewBufferString(`{"kind":"photo","note":"hull coating photo"}`)))
		if r.Code != http.StatusCreated {
			t.Fatalf("got %d body=%s", r.Code, r.Body.String())
		}
		r = httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/evidence/sf-301", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("got %d", r.Code)
		}
		var refs []EvidenceRef
		if err := json.Unmarshal(r.Body.Bytes(), &refs); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(refs) != 1 {
			t.Fatalf("len = %d", len(refs))
		}
	})
}

func TestReportAPI(t *testing.T) {
	t.Run("summary", func(t *testing.T) {
		h := newReportHandler(newFindingStore(), newOpsClock())
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/report/summary", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("got %d", r.Code)
		}
		var report SurveyReport
		if err := json.Unmarshal(r.Body.Bytes(), &report); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if report.Total != 2 {
			t.Fatalf("total = %d", report.Total)
		}
	})
}
