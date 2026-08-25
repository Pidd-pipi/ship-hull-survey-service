package main

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	sweepWorkers      = 4
	sweepStaleQueued  = 48 * time.Hour
	sweepStaleActive  = 7 * 24 * time.Hour
	scannerMaxHistory = 50
)

// SweepResult records one background sweep over the operations records.
type SweepResult struct {
	At       string
	Checked  int
	Closed   int
	Warnings []string
}

// Scanner periodically sweeps operations records and auto-closes stale
// queued work so the survey backlog does not pile up silently.
type Scanner struct {
	mu      sync.Mutex
	store   *OpsStore
	clock   OpsClock
	results []SweepResult
}

func newScanner(store *OpsStore, clock OpsClock) *Scanner {
	return &Scanner{store: store, clock: clock, results: []SweepResult{}}
}

// RunOnce inspects every operations record and closes queued records that
// have been waiting longer than the stale threshold. Records are dispatched
// to a small worker pool; the workers send their verdicts back over a
// channel and the coordinator applies the closures afterwards.
func (s *Scanner) RunOnce(ctx context.Context) SweepResult {
	items, err := s.store.List(ctx)
	if err != nil {
		return SweepResult{At: s.clock.Stamp(), Warnings: []string{"sweep aborted: " + err.Error()}}
	}

	type verdict struct {
		ID      string
		Close   bool
		Warning string
	}

	jobs := make(chan OpsRecord)
	verdicts := make(chan verdict, len(items))
	var wg sync.WaitGroup
	for i := 0; i < sweepWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				age := opsAge(s.clock.Now(), item.CreatedAt)
				switch item.Status {
				case OpsStatusQueued:
					if age >= sweepStaleQueued {
						verdicts <- verdict{ID: item.ID, Close: true, Warning: "stale queued record auto-closed"}
					}
				case OpsStatusActive:
					if age >= sweepStaleActive {
						verdicts <- verdict{ID: item.ID, Warning: "active record older than one week"}
					}
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, item := range items {
			select {
			case jobs <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(verdicts)
	}()

	result := SweepResult{At: s.clock.Stamp(), Checked: len(items), Warnings: []string{}}
	toClose := make([]string, 0, len(items))
	for v := range verdicts {
		if v.Close {
			toClose = append(toClose, v.ID)
			result.Closed++
		}
		if v.Warning != "" {
			result.Warnings = append(result.Warnings, v.ID+": "+v.Warning)
		}
	}
	for _, id := range toClose {
		if record, err := s.store.Get(ctx, id); err == nil {
			record.Status = OpsStatusClosed
			_ = s.store.Update(ctx, record, record.Revision)
		}
	}
	sort.Strings(result.Warnings)

	s.mu.Lock()
	s.results = append(s.results, result)
	if len(s.results) > scannerMaxHistory {
		s.results = s.results[len(s.results)-scannerMaxHistory:]
	}
	s.mu.Unlock()
	return result
}

// Start launches the periodic sweep loop. It stops when ctx is canceled.
func (s *Scanner) Start(ctx context.Context, interval time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.RunOnce(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Status returns the recent sweep history, newest first.
func (s *Scanner) Status() []SweepResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SweepResult, len(s.results))
	for i := range s.results {
		out[i] = s.results[len(s.results)-1-i]
	}
	return out
}

// newScannerHandler exposes the sweep workflow over HTTP.
func newScannerHandler(scanner *Scanner) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/ops/scanner/sweep", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		opsNoStore(w)
		opsJSON(w, http.StatusOK, scanner.RunOnce(r.Context()))
	})
	m.HandleFunc("/api/ops/scanner/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		opsNoStore(w)
		opsJSON(w, http.StatusOK, scanner.Status())
	})
	return m
}
