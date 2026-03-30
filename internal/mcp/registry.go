// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/client"
)

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(ctx context.Context, args json.RawMessage) (any, error)
}

type Registry struct {
	tools   map[string]Tool
	toolOrder []string
}

func NewRegistry(c client.Client, baseURL string) *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}
	r.registerAll(c, baseURL)
	return r
}

func (r *Registry) ListSchemas() []map[string]any {
	var tools []map[string]any
	for _, name := range r.toolOrder {
		t := r.tools[name]
		tools = append(tools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return tools
}

func (r *Registry) Call(id *json.RawMessage, name string, args json.RawMessage) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32601,
				Message: "Tool not found: " + name,
			},
		}, nil
	}

	ctx := context.Background()
	result, err := t.Handler(ctx, args)
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}, nil
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}, nil
}

func (r *Registry) register(t Tool) {
	r.tools[t.Name] = t
	r.toolOrder = append(r.toolOrder, t.Name)
}

func strPtr(s string) *string { return &s }

func requireString(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("Missing required argument: %s", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("Missing required argument: %s", key)
	}
	return s, nil
}

func requireFloat(args map[string]any, key string) (float64, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("Missing required argument: %s", key)
	}
	n, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("Missing required argument: %s", key)
	}
	return n, nil
}

func optString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func optFloat(args map[string]any, key string) float64 {
	v, ok := args[key]
	if !ok {
		return 0
	}
	n, ok := v.(float64)
	if !ok {
		return 0
	}
	return n
}

func optInt(args map[string]any, key string) int {
	return int(optFloat(args, key))
}

func optBool(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return b
}

func parseArgs(raw json.RawMessage) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	return args, nil
}
