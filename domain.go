package main

type SurveyFinding struct {
	ID       string `json:"id"`
	Vessel   string `json:"vessel"`
	Zone     string `json:"zone"`
	Finding  string `json:"finding"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
}
type StatusChange struct {
	Status string `json:"status"`
}
