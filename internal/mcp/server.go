// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func RunServer(ctx context.Context, registry *Registry, reader io.Reader, writer io.Writer) error {
	r := bufio.NewReader(reader)
	w := bufio.NewWriter(writer)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading stdin: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			resp := MCPErrorResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &MCPError{
					Code:    -32703,
					Message: "Failed to parse JSON: " + err.Error(),
				},
			}
			sendResponse(w, resp)
			continue
		}

		var resp any
		switch req.Method {
		case "initialize":
			resp = handleInitialize(req)
		case "notifications/initialized":
			continue
		case "tools/list":
			resp = handleToolsList(req, registry)
		case "tools/call":
			resp = handleToolsCall(req, registry)
		case "ping":
			resp = handlePing(req)
		default:
			resp = MCPErrorResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &MCPError{
					Code:    -32601,
					Message: "Method not found: " + req.Method,
				},
			}
		}

		sendResponse(w, resp)
	}
}

func sendResponse(writer *bufio.Writer, resp any) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	writer.Write(append(data, '\n'))
	writer.Flush()
}

func handleInitialize(req MCPRequest) any {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	protocolVersion := "2024-11-05"
	if err := json.Unmarshal(req.Params, &params); err == nil && params.ProtocolVersion != "" {
		protocolVersion = params.ProtocolVersion
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]string{
				"name":    "gitea-robot",
				"version": "1.0.0",
			},
		},
	}
}

func handleToolsList(req MCPRequest, registry *Registry) any {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": registry.ListSchemas(),
		},
	}
}

func handleToolsCall(req MCPRequest, registry *Registry) any {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments: " + err.Error(),
			},
		}
	}

	result, err := registry.Call(req.ID, params.Name, params.Arguments)
	if err != nil {
		return err
	}
	return result
}

func handlePing(req MCPRequest) any {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]string{},
	}
}
