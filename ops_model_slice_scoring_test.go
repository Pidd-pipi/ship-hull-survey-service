package main

import (
	"context"
	"testing"
)

func TestCreateIsolationFromCallerLabels(t *testing.T) {
	svc := newOpsService(opsSeed())
	labels := map[string]string{"site": "MV Aurora", "operator": "surveyor-x", "evidence": "photo-9"}
	rec := OpsRecord{ID: "hull-3001", Subject: "Bow thruster tunnel check", Owner: "surveyor-x", Status: OpsStatusQueued, Priority: OpsPriorityHigh, Labels: labels}
	if _, err := svc.Create(context.Background(), rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	labels["operator"] = "polluted"
	got, err := svc.Get(context.Background(), "hull-3001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Labels["operator"] == "polluted" {
		t.Fatal("stored record shares the caller's labels map")
	}
}

func TestOpsCloneIsDeep(t *testing.T) {
	rec := OpsRecord{ID: "hull-4001", Subject: "Keel scan", Owner: "surveyor-y", Status: OpsStatusQueued, Priority: OpsPriorityLow, Labels: map[string]string{"site": "MV T", "operator": "surveyor-y", "evidence": "s1"}}
	clone := rec.Clone()
	clone.Labels["operator"] = "tampered"
	if rec.Labels["operator"] == "tampered" {
		t.Fatal("Clone shares the labels map")
	}
}

func TestReportLinesStableAcrossCalls(t *testing.T) {
	clock := newOpsClock()
	first := buildSurveyReport([]SurveyFinding{
		{ID: "sf-a1", Zone: "port bow", Severity: "high", Status: "open"},
		{ID: "sf-a2", Zone: "aft bilge", Severity: "low", Status: "reviewed"},
	}, clock)
	buildSurveyReport([]SurveyFinding{
		{ID: "sf-b1", Zone: "cargo hold", Severity: "critical", Status: "open"},
	}, clock)
	// The first report must keep its own lines after another report is built.
	if len(first.Lines) != 2 {
		t.Fatalf("first report lines = %d, want 2 (data 串场)", len(first.Lines))
	}
	for _, line := range first.Lines {
		if line == "" || line[0] != 'p' && line[0] != 'a' {
			t.Fatalf("first report line polluted: %q", line)
		}
	}
}

func TestReportOpenCountsReviewed(t *testing.T) {
	report := buildSurveyReport([]SurveyFinding{
		{ID: "sf-o1", Zone: "port bow", Severity: "high", Status: "open"},
		{ID: "sf-r1", Zone: "aft bilge", Severity: "low", Status: "reviewed"},
		{ID: "sf-c1", Zone: "cargo hold", Severity: "critical", Status: "closed"},
	}, newOpsClock())
	if report.Open != 2 {
		t.Fatalf("Open = %d, want 2 (open + reviewed)", report.Open)
	}
}
