package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const maxEvidencePerFinding = 200

var evidenceSequence uint64

func newEvidenceID() string { return fmt.Sprintf("ev-%06d", atomic.AddUint64(&evidenceSequence, 1)) }

// EvidenceRef is one piece of inspection evidence attached to a survey finding.
type EvidenceRef struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Note string `json:"note"`
	At   string `json:"at"`
}

// EvidenceStore keeps the evidence refs attached to survey findings.
type EvidenceStore struct {
	mu    sync.RWMutex
	items map[string][]EvidenceRef
}

func newEvidenceStore() *EvidenceStore { return &EvidenceStore{items: map[string][]EvidenceRef{}} }

func (s *EvidenceStore) Add(findingID, kind, note string) (EvidenceRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	refs := s.items[findingID]
	if len(refs) >= maxEvidencePerFinding {
		return EvidenceRef{}, errors.New("evidence limit reached for finding " + findingID)
	}
	ref := EvidenceRef{ID: newEvidenceID(), Kind: kind, Note: note, At: time.Now().UTC().Format(time.RFC3339Nano)}
	s.items[findingID] = append(refs, ref)
	return ref, nil
}

func (s *EvidenceStore) For(findingID string) []EvidenceRef {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]EvidenceRef(nil), s.items[findingID]...)
}

// newEvidenceHandler exposes evidence attachments for survey findings.
func newEvidenceHandler(findings *FindingStore, evidence *EvidenceStore) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/evidence/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/evidence/"), "/")
		if path == "" || strings.Contains(path, "/") {
			opsJSON(w, http.StatusNotFound, map[string]string{"error": "survey finding not found"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			opsJSON(w, http.StatusOK, evidence.For(path))
		case http.MethodPost:
			handleEvidenceAdd(w, r, findings, evidence, path)
		default:
			opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	})
	return m
}

func handleEvidenceAdd(w http.ResponseWriter, r *http.Request, findings *FindingStore, evidence *EvidenceStore, id string) {
	var body struct {
		Kind string `json:"kind"`
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if _, err := findings.get(id); err != nil {
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "survey finding not found"})
		return
	}
	if err := validateEvidenceKind(body.Kind); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ref, err := evidence.Add(id, body.Kind, body.Note)
	if err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusCreated, ref)
}
