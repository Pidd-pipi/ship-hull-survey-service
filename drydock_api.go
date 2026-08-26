package main

import (
	"encoding/json"
	"net/http"
)

// newDryDockHandler exposes the dry-dock planner over HTTP.
func newDryDockHandler(dock *DryDock) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/drydock/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		handleDryDockCheck(w, r, dock)
	})
	m.HandleFunc("/api/drydock/book", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		handleDryDockBook(w, r, dock)
	})
	return m
}

func handleDryDockCheck(w http.ResponseWriter, r *http.Request, dock *DryDock) {
	var requests []DockRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	opsJSON(w, http.StatusOK, dock.CheckAvailability(r.Context(), requests))
}

func handleDryDockBook(w http.ResponseWriter, r *http.Request, dock *DryDock) {
	var req DockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	res, err := dock.Book(req)
	if err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusCreated, res)
}
