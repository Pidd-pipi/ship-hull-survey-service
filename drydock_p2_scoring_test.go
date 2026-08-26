package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func dockRequests(n int) []DockRequest {
	out := make([]DockRequest, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, DockRequest{
			Vessel: fmt.Sprintf("vessel-%02d", i),
			Berth:  "B1",
			Start:  "2026-09-01T08:00:00Z",
			Hours:  24,
		})
	}
	return out
}

func TestDryDockCheckAllRequests(t *testing.T) {
	dock := newDryDock(newOpsClock())
	results := dock.CheckAvailability(context.Background(), dockRequests(20))
	if len(results) != 20 {
		t.Fatalf("results = %d, want 20", len(results))
	}
	for _, res := range results {
		if !res.OK {
			t.Fatalf("unexpected failure: %+v", res)
		}
	}
}

func TestDryDockConcurrentChecks(t *testing.T) {
	dock := newDryDock(newOpsClock())
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 20; j++ {
				dock.CheckAvailability(context.Background(), dockRequests(10))
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestDryDockCheckViaHTTP(t *testing.T) {
	dock := newDryDock(newOpsClock())
	h := newDryDockHandler(dock)
	body, _ := json.Marshal(dockRequests(12))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/drydock/check", bytes.NewReader(body)))
	if r.Code != http.StatusOK {
		t.Fatalf("got %d", r.Code)
	}
	var results []DockResult
	if err := json.Unmarshal(r.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(results) != 12 {
		t.Fatalf("http results = %d, want 12", len(results))
	}
}

func TestDryDockBookConflictRejected(t *testing.T) {
	dock := newDryDock(newOpsClock())
	first := DockRequest{Vessel: "v1", Berth: "B2", Start: "2026-09-01T08:00:00Z", Hours: 24}
	overlap := DockRequest{Vessel: "v2", Berth: "B2", Start: "2026-09-01T12:00:00Z", Hours: 24}
	if _, err := dock.Book(first); err != nil {
		t.Fatalf("first book: %v", err)
	}
	if _, err := dock.Book(overlap); err == nil {
		t.Fatal("overlapping booking was accepted")
	}
}

func TestDryDockBerthsReturnsCopy(t *testing.T) {
	dock := newDryDock(newOpsClock())
	if _, err := dock.Book(DockRequest{Vessel: "v1", Berth: "B3", Start: "2026-09-01T08:00:00Z", Hours: 24}); err != nil {
		t.Fatalf("book: %v", err)
	}
	slots := dock.Berths("B3")
	if len(slots) > 0 {
		slots[0].From = slots[0].From.Add(-48 * time.Hour)
	}
	again := dock.Berths("B3")
	if len(again) > 0 && again[0].From.Before(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("Berths() leaked internal slice")
	}
}

func TestDryDockBookConflictViaHTTP(t *testing.T) {
	dock := newDryDock(newOpsClock())
	h := newDryDockHandler(dock)
	first := `{"vessel":"v1","berth":"B2","start":"2026-09-01T08:00:00Z","hours":24}`
	overlap := `{"vessel":"v2","berth":"B2","start":"2026-09-01T12:00:00Z","hours":24}`
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/drydock/book", bytes.NewBufferString(first)))
	if r.Code != http.StatusCreated {
		t.Fatalf("first book got %d body=%s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/drydock/book", bytes.NewBufferString(overlap)))
	if r.Code != http.StatusConflict {
		t.Fatalf("overlap book got %d, want 409 (body=%s)", r.Code, r.Body.String())
	}
}
