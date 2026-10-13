package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestRoutes(t *testing.T) {
	r := newRouter(prometheus.NewRegistry())

	tests := []struct {
		path        string
		wantStatus  int
		wantBody    string
		wantType    string
		wantHeaders map[string]string
	}{
		{path: "/healthz", wantStatus: http.StatusOK, wantBody: "hi."},
		{path: "/", wantStatus: http.StatusTemporaryRedirect, wantHeaders: map[string]string{"Location": "/2024-12-14"}},
		{path: "/2024-12-12", wantStatus: http.StatusOK, wantBody: "Be yourself; everyone else is already taken."},
		{path: "/1999-01-01", wantStatus: http.StatusNotFound},
		{path: "/2024-13-45", wantStatus: http.StatusNotFound},
		{path: "/about", wantStatus: http.StatusOK},
		{path: "/feed.rss", wantStatus: http.StatusOK, wantType: "application/rss+xml", wantBody: "https://distraction.today/2024-12-13"},
		{path: "/feed.atom", wantStatus: http.StatusOK, wantType: "application/atom+xml", wantBody: "https://distraction.today/2024-12-14"},
		{path: "/metrics", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tt.path, rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("GET %s body missing %q", tt.path, tt.wantBody)
			}
			if tt.wantType != "" && !strings.HasPrefix(rec.Header().Get("Content-Type"), tt.wantType) {
				t.Errorf("GET %s Content-Type = %q, want %q", tt.path, rec.Header().Get("Content-Type"), tt.wantType)
			}
			for k, v := range tt.wantHeaders {
				if got := rec.Header().Get(k); got != v {
					t.Errorf("GET %s %s = %q, want %q", tt.path, k, got, v)
				}
			}
		})
	}
}

func TestGenerateFeed(t *testing.T) {
	feed, err := generateFeed()
	if err != nil {
		t.Fatalf("generateFeed() error = %v", err)
	}
	if len(feed.Items) != 3 {
		t.Errorf("generateFeed() items = %d, want 3", len(feed.Items))
	}
}
