package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func newRouter(store *FindingStore) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/healthz", healthHandler("ship-hull-survey-service"))
	m.HandleFunc("/api/findings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/findings" {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, 200, store.list())
	})
	m.HandleFunc("/api/findings/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/findings/")
		switch {
		case strings.HasSuffix(path, "/status"):
			if r.Method != http.MethodPost {
				writeJSON(w, 405, map[string]string{"error": "method not allowed"})
				return
			}
			id := strings.TrimSuffix(path, "/status")
			if id == "" || strings.Contains(id, "/") {
				writeJSON(w, 404, map[string]string{"error": "survey finding not found"})
				return
			}
			var c StatusChange
			if json.NewDecoder(r.Body).Decode(&c) != nil || validateSurveyStatus(c.Status) != nil {
				writeJSON(w, 400, map[string]string{"error": "valid status is required"})
				return
			}
			v, e := store.changeStatus(id, c.Status)
			if errors.Is(e, errFindingNotFound) {
				writeJSON(w, 404, map[string]string{"error": e.Error()})
				return
			}
			writeJSON(w, 200, v)
		case strings.Contains(path, "/"):
			writeJSON(w, 404, map[string]string{"error": "survey finding not found"})
		default:
			switch r.Method {
			case http.MethodGet:
				v, e := store.get(path)
				if errors.Is(e, errFindingNotFound) {
					writeJSON(w, 404, map[string]string{"error": e.Error()})
					return
				}
				writeJSON(w, 200, v)
			case http.MethodDelete:
				if e := store.remove(path); errors.Is(e, errFindingNotFound) {
					writeJSON(w, 404, map[string]string{"error": e.Error()})
					return
				}
				writeJSON(w, 200, map[string]string{"status": "removed"})
			default:
				writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			}
		}
	})
	return m
}
