// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/client"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/config"
)

func newTestClientAndServer(handler http.HandlerFunc) (client.Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	cfg := &config.Config{
		BaseURL:          srv.URL,
		Token:            "test-token",
		MaxResponseBytes: 1024 * 1024,
	}
	return client.NewHTTPClient(cfg), srv
}

func TestGetTriage(t *testing.T) {
	expected := TriageResult{
		QuickRef: QuickRef{Total: 10, Open: 5, Blocked: 2, Ready: 3},
		Recommendations: []Recommendation{
			{Index: 1, Title: "Fix auth", PageRank: 0.85},
			{Index: 2, Title: "Add tests", PageRank: 0.42},
		},
	}

	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("owner") != "terraphim" || r.URL.Query().Get("repo") != "gitea" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		data, _ := json.Marshal(expected)
		w.Write(data)
	})
	defer srv.Close()

	result, err := GetTriage(context.Background(), c, srv.URL, "terraphim", "gitea")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.QuickRef.Total != 10 {
		t.Errorf("expected total 10, got %.0f", result.QuickRef.Total)
	}
	if len(result.Recommendations) != 2 {
		t.Fatalf("expected 2 recommendations, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].Title != "Fix auth" {
		t.Errorf("expected 'Fix auth', got %s", result.Recommendations[0].Title)
	}
}

func TestGetReady(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"index":1,"title":"Ready task"}]`))
	})
	defer srv.Close()

	data, err := GetReady(context.Background(), c, srv.URL, "terraphim", "gitea")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty response")
	}
}

func TestGetGraph(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"nodes":[],"edges":[]}`))
	})
	defer srv.Close()

	data, err := GetGraph(context.Background(), c, srv.URL, "terraphim", "gitea")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty response")
	}
}

func TestAddDependency(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()

	err := AddDependency(context.Background(), c, srv.URL, "terraphim", "gitea", 2, 1, "blocks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloseIssue(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.Write([]byte(`{"number":42,"state":"closed"}`))
	})
	defer srv.Close()

	issue, err := CloseIssue(context.Background(), c, srv.URL, "terraphim", "gitea", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.State != "closed" {
		t.Errorf("expected state 'closed', got %s", issue.State)
	}
}

func TestCreateIssue(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"number":1,"title":"Bug fix","state":"open"}`))
	})
	defer srv.Close()

	issue, err := CreateIssue(context.Background(), c, srv.URL, "terraphim", "gitea", CreateIssueOpts{
		Title: "Bug fix",
		Body:  "Fix the thing",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Title != "Bug fix" {
		t.Errorf("expected 'Bug fix', got %s", issue.Title)
	}
}

func TestListIssues(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("expected state=open, got %s", r.URL.Query().Get("state"))
		}
		w.Write([]byte(`[{"number":1,"title":"Test","state":"open"}]`))
	})
	defer srv.Close()

	issues, err := ListIssues(context.Background(), c, srv.URL, "terraphim", "gitea", ListIssueOpts{State: "open", Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Title != "Test" {
		t.Errorf("expected 'Test', got %s", issues[0].Title)
	}
}

func TestResolveLabels(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":1,"name":"bug","color":"#FF0000"},{"id":2,"name":"feature","color":"#00FF00"}]`))
	})
	defer srv.Close()

	ids, err := ResolveLabels(context.Background(), c, srv.URL, "terraphim", "gitea", []string{"bug", "feature"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
	if ids[0] != 1 || ids[1] != 2 {
		t.Errorf("expected [1,2], got %v", ids)
	}
}

func TestURLEncoding(t *testing.T) {
	c, srv := newTestClientAndServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	_, err := ListIssues(context.Background(), c, srv.URL, "foo/bar", "repo with spaces", ListIssueOpts{State: "open", Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error with special chars: %v", err)
	}
}
