package main

import (
	"context"
	"sort"
	"sync"
	"time"
)

const dockCheckWorkers = 4

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
	for i := 0; i < dockCheckWorkers; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()
			for req := range jobs {
				results <- d.checkOne(req)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for i := 0; i < len(requests)/2; i++ {
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
	for _, slot := range d.bookings[req.Berth] {
		if start.Before(slot.To) && end.After(slot.From) {
			return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: false, Reason: "berth occupied"}
		}
	}
	return DockResult{Vessel: req.Vessel, Berth: req.Berth, Start: req.Start, OK: true}
}

// Book reserves the berth window for a vessel.
func (d *DryDock) Book(req DockRequest) (DockResult, error) {
	res := d.checkOne(req)
	start, _ := time.Parse(time.RFC3339, req.Start)
	end := start.Add(time.Duration(req.Hours) * time.Hour)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bookings[req.Berth] = append(d.bookings[req.Berth], DockSlot{Berth: req.Berth, From: start, To: end})
	return res, nil
}

// Berths returns the reserved windows for a berth.
func (d *DryDock) Berths(berth string) []DockSlot {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]DockSlot(nil), d.bookings[berth]...)
}
