package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSurveyHTTP(t *testing.T) {
	h := newRouter(newFindingStore())
	cases := []struct {
		name, path, body string
		code             int
	}{{"collection", "/api/findings", "", 200}, {"close", "/api/findings/sf-301/status", `{"status":"closed"}`, 200}, {"invalid", "/api/findings/sf-301/status", `{"status":"unknown"}`, 400}, {"missing", "/api/findings/nope/status", `{"status":"open"}`, 404}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			method := http.MethodGet
			if tc.body != "" {
				method = http.MethodPost
			}
			h.ServeHTTP(r, httptest.NewRequest(method, tc.path, bytes.NewBufferString(tc.body)))
			if r.Code != tc.code {
				t.Fatalf("got %d", r.Code)
			}
		})
	}
}
