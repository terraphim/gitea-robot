// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewRegistry_HasAllTools(t *testing.T) {
	r := NewRegistry(nil, "http://localhost:3000")
	tools := r.ListSchemas()

	if len(tools) != 20 {
		t.Errorf("expected 20 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		n, _ := tool["name"].(string)
		names[n] = true
	}

	expected := []string{
		"triage", "ready", "graph", "add_dep",
		"list_labels", "list_pulls", "create_pull", "merge_pull",
		"view_issue", "view_pull", "create_label", "create_repo",
		"create_release", "list_repos", "fork_repo",
		"list_issues", "create_issue", "comment", "close_issue", "edit_issue",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestRegistry_Call_UnknownTool(t *testing.T) {
	r := NewRegistry(nil, "http://localhost:3000")
	id := json.RawMessage(`1`)

	resp, err := r.Call(&id, "nonexistent", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	errResp, ok := resp.(MCPErrorResponse)
	if !ok {
		t.Fatalf("expected MCPErrorResponse, got %T", resp)
	}
	if errResp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", errResp.Error.Code)
	}
}

func TestRegistry_Call_MissingRequiredArgs(t *testing.T) {
	r := NewRegistry(nil, "http://localhost:3000")
	id := json.RawMessage(`1`)

	tests := []struct {
		tool string
		args string
	}{
		{"triage", `{"repo":"gitea"}`},
		{"ready", `{"repo":"gitea"}`},
		{"graph", `{"repo":"gitea"}`},
		{"add_dep", `{"repo":"gitea","issue":2,"blocks":1}`},
		{"view_issue", `{"repo":"gitea","index":1}`},
		{"create_issue", `{"repo":"gitea","title":"test"}`},
	}

	for _, tt := range tests {
		resp, _ := r.Call(&id, tt.tool, json.RawMessage(tt.args))
		errResp, ok := resp.(MCPErrorResponse)
		if !ok {
			t.Errorf("tool %s: expected MCPErrorResponse, got %T", tt.tool, resp)
			continue
		}
		if !strings.Contains(errResp.Error.Message, "owner") {
			t.Errorf("tool %s: expected error about owner, got: %s", tt.tool, errResp.Error.Message)
		}
	}
}

func TestRegistry_Call_InvalidJSON(t *testing.T) {
	r := NewRegistry(nil, "http://localhost:3000")
	id := json.RawMessage(`1`)

	resp, err := r.Call(&id, "triage", json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	errResp, ok := resp.(MCPErrorResponse)
	if !ok {
		t.Fatalf("expected MCPErrorResponse, got %T", resp)
	}
}

func jsonRawMessagePtr(s string) *json.RawMessage {
	raw := json.RawMessage(s)
	return &raw
}
