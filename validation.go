package main

import "fmt"

func validateSurveyStatus(s string) error {
	switch s {
	case "open", "reviewed", "closed":
		return nil
	default:
		return fmt.Errorf("status must be open, reviewed, or closed")
	}
}

func validateEvidenceKind(kind string) error {
	switch kind {
	case "photo", "note":
		return nil
	default:
		return fmt.Errorf("evidence kind must be photo, scan, or note")
	}
}
