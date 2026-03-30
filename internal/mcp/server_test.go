// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func ctx() context.Context {
	return context.Background()
}

func TestMCPServer_Initialize(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	reg := NewRegistry(nil, "http://localhost:3000")

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunServer(ctx(), reg, strings.NewReader(input), bufio.NewWriter(&out))
	}()
	<-done

	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nraw: %s", err, out.String())
	}
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %v", resp["jsonrpc"])
	}
	result, _ := resp["result"].(map[string]any)
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocol 2024-11-05, got %v", result["protocolVersion"])
	}
}

func TestMCPServer_Ping(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n"
	reg := NewRegistry(nil, "http://localhost:3000")

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunServer(ctx(), reg, strings.NewReader(input), bufio.NewWriter(&out))
	}()
	<-done

	var resp map[string]any
	json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp)
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %v", resp["jsonrpc"])
	}
}

func TestMCPServer_InvalidJSON(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":4,"method":}` + "\n"
	reg := NewRegistry(nil, "http://localhost:3000")

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunServer(ctx(), reg, strings.NewReader(input), bufio.NewWriter(&out))
	}()
	<-done

	var resp map[string]any
	json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp)
	errObj, _ := resp["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); code != -32703 {
		t.Errorf("expected code -32703, got %v", errObj["code"])
	}
}

func TestMCPServer_UnknownMethod(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":5,"method":"unknown"}` + "\n"
	reg := NewRegistry(nil, "http://localhost:3000")

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunServer(ctx(), reg, strings.NewReader(input), bufio.NewWriter(&out))
	}()
	<-done

	var resp map[string]any
	json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp)
	errObj, _ := resp["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); code != -32601 {
		t.Errorf("expected code -32601, got %v", errObj["code"])
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}` + "\n"
	reg := NewRegistry(nil, "http://localhost:3000")

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunServer(ctx(), reg, strings.NewReader(input), bufio.NewWriter(&out))
	}()
	<-done

	var resp map[string]any
	json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp)
	result, _ := resp["result"].(map[string]any)
	tools, _ := result["tools"].([]any)
	if len(tools) != 20 {
		t.Errorf("expected 20 tools, got %d", len(tools))
	}
}
