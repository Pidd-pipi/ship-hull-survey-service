package main

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

const dockCheckWorkers = 4

var (
	errDockInvalid  = errors.New("invalid docking request")
	errDockConflict = errors.New("berth occupied")
)

// DockSlot is one reserved time window on a dry-dock berth.
type DockSlot struct {
	Berth string
	From  time.Time
	To    time.Time
}

// DockRequest asks whether a vessel can be placed on a berth for a window.
type DockRequest struct {
	Vessel string `json:"vessel"`
	Berth  string `json:"berth"`
	Start  string `json:"start"`
	Hours  int    `json:"hours"`
}

// DockResult reports the availability verdict for one request.
type DockResult struct {
	Vessel string `json:"vessel"`
	Berth  string `json:"berth"`
	Start  string `json:"start"`
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

// DryDock tracks berth reservations and checks availability for survey
// vessels that need to be hauled out for hull inspection.
type DryDock struct {
	mu       sync.Mutex
	bookings map[string][]DockSlot
	clock    OpsClock
}

func newDryDock(clock OpsClock) *DryDock {
	return &DryDock{bookings: map[string][]DockSlot{}, clock: clock}
}

// CheckAvailability runs overlap checks for a batch of docking requests
// across a small worker pool and returns one verdict per request.
func (d *DryDock) CheckAvailability(ctx context.Context, requests []DockRequest) []DockResult {
	jobs := make(chan DockRequest)
	results := make(chan DockResult, len(requests))
	var wg sync.WaitGroup
	// Add the worker count before spawning so wg.Wait cannot return before the
	// workers have registered — otherwise close(results) races with the
	// workers sending on results (send on closed channel).
	wg.Add(dockCheckWorkers)
	for i := 0; i < dockCheckWorkers; i++ {
		go func() {
			defer wg.Done()
			for req := range jobs {
				results <- d.checkOne(req)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for i := 0; i < len(requests); i++ {
			select {
			case jobs <- requests[i]:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]DockResult, 0, len(requests))
	for res := range results {
		out = append(out, res)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Vessel < out[j].Vessel })
	return out
}

func (d *DryDock) checkOne(req DockRequest) DockResult {
	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "invalid start time"}
	}
	if req.Hours < 1 || req.Hours > 72 {
		return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "invalid hours"}
	}
	end := start.Add(time.Duration(req.Hours) * time.Hour)
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.overlapsLocked(req.Berth, start, end) {
		return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "berth occupied"}
	}
	return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: true}
}

// overlapsLocked reports whether [start,end) collides with any booked slot
// on the berth. The caller must hold d.mu.
func (d *DryDock) overlapsLocked(berth string, start, end time.Time) bool {
	for _, slot := range d.bookings[berth] {
		if start.Before(slot.To) && end.After(slot.From) {
			return true
		}
	}
	return false
}

// Book reserves the berth window for a vessel. The validation, overlap check
// and append run under a single lock so concurrent bookings cannot double-book
// the same window. An occupied berth returns errDockConflict; bad input
// returns errDockInvalid.
func (d *DryDock) Book(req DockRequest) (DockResult, error) {
	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "invalid start time"}, errDockInvalid
	}
	if req.Hours < 1 || req.Hours > 72 {
		return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "invalid hours"}, errDockInvalid
	}
	end := start.Add(time.Duration(req.Hours) * time.Hour)
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.overlapsLocked(req.Berth, start, end) {
		return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "berth occupied"}, errDockConflict
	}
	d.bookings[req.Berth] = append(d.bookings[req.Berth], DockSlot{Berth: req.Berth, From: start, To: end})
	return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: true}, nil
}

// Berths returns the reserved windows for a berth.
func (d *DryDock) Berths(berth string) []DockSlot {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]DockSlot(nil), d.bookings[berth]...)
}
