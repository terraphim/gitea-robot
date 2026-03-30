// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireOwnerRepo(t *testing.T) {
	if err := RequireOwnerRepo("foo", "bar"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := RequireOwnerRepo("", "bar"); err == nil {
		t.Error("expected error for empty owner")
	}
	if err := RequireOwnerRepo("foo", ""); err == nil {
		t.Error("expected error for empty repo")
	}
}

func TestSplitLabelNames(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"bug", []string{"bug"}},
		{"bug,feature", []string{"bug", "feature"}},
		{" bug , feature ", []string{"bug", "feature"}},
	}
	for _, tt := range tests {
		got := SplitLabelNames(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("SplitLabelNames(%q) = %v, want %v", tt.input, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("SplitLabelNames(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestReadBody_FromFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "body.txt")
	os.WriteFile(f, []byte("hello world"), 0644)

	got, err := ReadBody("", f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestReadBody_FromFlag(t *testing.T) {
	got, err := ReadBody("flag body", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "flag body" {
		t.Errorf("expected 'flag body', got %q", got)
	}
}

func TestReadBody_Empty(t *testing.T) {
	got, err := ReadBody("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
