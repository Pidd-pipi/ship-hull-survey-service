package main

import (
	"net/http"
	"sort"
	"strings"
)

// SurveyReport summarizes the current survey findings for the vessel fleet.
type SurveyReport struct {
	GeneratedAt string
	Total       int
	Open        int
	BySeverity  map[string]int
	ByZone      map[string]int
	Lines       []string
}

// buildSurveyReport aggregates findings into a human readable summary. The
// returned lines are sorted by zone then severity so the report is stable.
func buildSurveyReport(findings []SurveyFinding, clock OpsClock) SurveyReport {
	report := SurveyReport{
		GeneratedAt: clock.Stamp(),
		BySeverity:  map[string]int{},
		ByZone:      map[string]int{},
	}
	for _, finding := range findings {
		report.Total++
		if finding.Status == "open" || finding.Status == "reviewed" {
			report.Open++
		}
		report.BySeverity[finding.Severity]++
		report.ByZone[finding.Zone]++
	}
	zones := make([]string, 0, len(report.ByZone))
	for zone := range report.ByZone {
		zones = append(zones, zone)
	}
	sort.Strings(zones)
	for _, zone := range zones {
		report.Lines = append(report.Lines, formatZoneLine(zone, report.ByZone[zone], report.BySeverity))
	}
	return report
}

// formatZoneLine builds one report line for a survey zone.
func formatZoneLine(zone string, count int, severity map[string]int) string {
	parts := make([]string, 0, 3)
	for _, level := range []string{"critical", "high", "medium", "low"} {
		if n := severity[level]; n > 0 {
			parts = append(parts, level+" "+itoa(n))
		}
	}
	body := strings.Join(parts, ", ")
	if body == "" {
		body = "no severity breakdown"
	}
	return zone + " (" + itoa(count) + "): " + body
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	out := string(digits[index:])
	if negative {
		out = "-" + out
	}
	return out
}

// newReportHandler exposes the survey summary report.
func newReportHandler(findings *FindingStore, clock OpsClock) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/report/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		opsJSON(w, http.StatusOK, buildSurveyReport(findings.list(), clock))
	})
	return m
}
