package main

import (
	"errors"
	"sync"
)

var errFindingNotFound = errors.New("survey finding not found")

type FindingStore struct {
	mu    sync.RWMutex
	items map[string]SurveyFinding
}

func newFindingStore() *FindingStore {
	return &FindingStore{items: map[string]SurveyFinding{"sf-301": {ID: "sf-301", Vessel: "MV Aurora", Zone: "port bow", Finding: "coating blistering", Severity: "medium", Status: "open"}, "sf-302": {ID: "sf-302", Vessel: "MV Aurora", Zone: "aft bilge", Finding: "weld seam mark", Severity: "low", Status: "reviewed"}}}
}
func (s *FindingStore) list() []SurveyFinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var o []SurveyFinding
	for _, v := range s.items {
		o = append(o, v)
	}
	return o
}
func (s *FindingStore) get(id string) (SurveyFinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[id], nil
}
func (s *FindingStore) changeStatus(id, status string) (SurveyFinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[id]
	if !ok {
		return SurveyFinding{}, errFindingNotFound
	}
	v.Status = status
	s.items[id] = v
	return v, nil
}
func (s *FindingStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return errFindingNotFound
	}
	delete(s.items, id)
	return nil
}
