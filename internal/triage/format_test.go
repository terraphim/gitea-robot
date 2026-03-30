// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package triage

import (
	"bytes"
	"testing"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/gitea"
)

func TestFormatMarkdown(t *testing.T) {
	result := &gitea.TriageResult{
		QuickRef: gitea.QuickRef{Total: 10, Open: 5, Blocked: 2, Ready: 3},
		Recommendations: []gitea.Recommendation{
			{Index: 1, Title: "Fix auth", PageRank: 0.85},
			{Index: 2, Title: "Add tests", PageRank: 0.42},
		},
	}

	var buf bytes.Buffer
	FormatMarkdown(result, &buf)

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("## Triage Report")) {
		t.Error("expected '## Triage Report' header")
	}
	if !bytes.Contains(buf.Bytes(), []byte("**Stats:** Total: 10")) {
		t.Error("expected stats line")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Fix auth")) {
		t.Error("expected recommendation 'Fix auth'")
	}
	if !bytes.Contains(buf.Bytes(), []byte("0.8500")) {
		t.Errorf("expected PageRank score, got: %s", output)
	}
}

func TestFormatMarkdown_EmptyRecommendations(t *testing.T) {
	result := &gitea.TriageResult{
		QuickRef:       gitea.QuickRef{Total: 0, Open: 0, Blocked: 0, Ready: 0},
		Recommendations: nil,
	}

	var buf bytes.Buffer
	FormatMarkdown(result, &buf)

	if bytes.Contains(buf.Bytes(), []byte("### Top Recommendations")) {
		t.Error("should not show recommendations section when empty")
	}
}

func TestFormatMarkdown_LimitsTo10(t *testing.T) {
	recs := make([]gitea.Recommendation, 15)
	for i := range recs {
		recs[i] = gitea.Recommendation{Index: float64(i + 1), Title: "Issue {i}", PageRank: 0.1}
	}
	result := &gitea.TriageResult{
		QuickRef:       gitea.QuickRef{Total: 15, Open: 15},
		Recommendations: recs,
	}

	var buf bytes.Buffer
	FormatMarkdown(result, &buf)

	count := bytes.Count(buf.Bytes(), []byte("**#"))
	if count != 10 {
		t.Errorf("expected 10 recommendations, got %d", count)
	}
}
