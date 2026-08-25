package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// opsSeed returns the baseline operations records used by the service.
func opsSeed() []OpsRecord {
	return []OpsRecord{
		{ID: "hull-1001", Subject: "Port bow coating inspection", Owner: "surveyor-lin", Status: OpsStatusQueued, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "MV Aurora", "operator": "surveyor-lin", "evidence": "photo-1"}, Revision: 1},
		{ID: "hull-1002", Subject: "Aft bilge weld review", Owner: "surveyor-zhao", Status: OpsStatusActive, Priority: OpsPriorityNormal, Labels: map[string]string{"site": "MV Aurora", "operator": "surveyor-zhao", "evidence": "photo-2"}, Revision: 1},
		{ID: "hull-1003", Subject: "Cargo hold floor corrosion", Owner: "surveyor-lin", Status: OpsStatusPaused, Priority: OpsPriorityCritical, Labels: map[string]string{"site": "MV Orion", "operator": "surveyor-lin", "evidence": "photo-3"}, Revision: 1},
	}
}

// newOpsHandler exposes the operations workflow over HTTP.
func newOpsHandler(service *OpsService) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/ops/records", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleOpsSearch(w, r, service)
		case http.MethodPost:
			handleOpsCreate(w, r, service)
		default:
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	})
	m.HandleFunc("/api/ops/records/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/ops/records/")
		switch {
		case strings.HasSuffix(path, "/transition"):
			if r.Method != http.MethodPost {
				opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			handleOpsTransition(w, r, service, strings.TrimSuffix(path, "/transition"))
		case strings.HasSuffix(path, "/audit"):
			if r.Method != http.MethodGet {
				opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			handleOpsAudit(w, r, service, strings.TrimSuffix(path, "/audit"))
		case strings.Contains(path, "/"):
			opsJSON(w, http.StatusNotFound, map[string]string{"error": "operations record not found"})
		default:
			handleOpsGet(w, r, service, path)
		}
	})
	m.HandleFunc("/api/ops/snapshot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		opsNoStore(w)
		opsJSON(w, http.StatusOK, service.Snapshot())
	})
	return m
}

// opsStatusForCode maps the operations error code back to an HTTP status.
func opsStatusForCode(code string) int {
	switch code {
	case "not_found":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	case "invalid", "transition", "policy":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func handleOpsSearch(w http.ResponseWriter, r *http.Request, service *OpsService) {
	q := OpsQuery{
		Subject:  r.URL.Query().Get("subject"),
		Status:   OpsStatus(r.URL.Query().Get("status")),
		Priority: OpsPriority(r.URL.Query().Get("priority")),
		Owner:    r.URL.Query().Get("owner"),
	}
	if v := r.URL.Query().Get("page"); v != "" {
		q.Page, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		q.PageSize, _ = strconv.Atoi(v)
	}
	page, err := service.Search(r.Context(), q)
	if err != nil {
		opsJSON(w, opsStatusForCode(opsCode(err)), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, page)
}

func handleOpsGet(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	record, err := service.Get(r.Context(), id)
	if err != nil {
		opsJSON(w, opsStatusForCode(opsCode(err)), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func handleOpsCreate(w http.ResponseWriter, r *http.Request, service *OpsService) {
	var record OpsRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	created, err := service.Create(r.Context(), record)
	if err != nil {
		opsJSON(w, opsStatusForCode(opsCode(err)), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusCreated, created)
}

func handleOpsTransition(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	var body struct {
		Expected int       `json:"expected"`
		Status   OpsStatus `json:"status"`
		Actor    string    `json:"actor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	actor := body.Actor
	if actor == "" {
		actor = opsActorFromRequest(r)
	}
	record, err := service.Transition(r.Context(), id, body.Expected, body.Status, actor)
	if err != nil {
		opsJSON(w, opsStatusForCode(opsCode(err)), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func handleOpsAudit(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	opsNoStore(w)
	events := service.Audit(id)
	if len(events) == 0 {
		opsJSON(w, http.StatusOK, []OpsEvent{})
		return
	}
	opsJSON(w, http.StatusOK, events)
}
