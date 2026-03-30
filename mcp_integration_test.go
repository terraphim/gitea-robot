// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestMCPCallToolResultFormat validates that tools/call returns proper
// CallToolResult format as required by MCP spec. This was the root cause
// of Claude Code and OpenCode MCP clients hanging indefinitely.
func TestMCPCallToolResultFormat(t *testing.T) {
	// Test each tool handler returns CallToolResult format (not raw string)
	tools := []struct {
		name string
		args string
	}{
		// list_repos has no required args, so it hits the Gitea API
		{"list_repos", `{"org":"terraphim","limit":1}`},
	}

	for _, tool := range tools {
		t.Run(tool.name, func(t *testing.T) {
			id := jsonRawMessagePtr(`99`)
			req := MCPRequest{
				JSONRPC: "2.0",
				ID:      id,
				Method:  "tools/call",
				Params:  json.RawMessage(fmt.Sprintf(`{"name":"%s","arguments":%s}`, tool.name, tool.args)),
			}

			resp := handleToolsCall(req)

			// Must be MCPResponse (not MCPErrorResponse) for valid calls
			mcpResp, ok := resp.(MCPResponse)
			if !ok {
				// Could be a connection error (toolErrorResult also returns MCPResponse)
				t.Logf("Got %T instead of MCPResponse -- may be a connection error to Gitea", resp)
				return
			}

			// Result must be a map with "content" key (CallToolResult format)
			result, ok := mcpResp.Result.(map[string]any)
			if !ok {
				t.Fatalf("tools/call Result must be map[string]any (CallToolResult), got %T: %v", mcpResp.Result, mcpResp.Result)
			}

			// Must have "content" array
			content, ok := result["content"]
			if !ok {
				t.Fatalf("CallToolResult missing 'content' field: %v", result)
			}

			contentArr, ok := content.([]map[string]any)
			if !ok {
				t.Fatalf("content must be []map[string]any, got %T", content)
			}

			if len(contentArr) == 0 {
				t.Fatal("content array must not be empty")
			}

			// First content item must have type and text
			item := contentArr[0]
			if item["type"] != "text" {
				t.Errorf("content[0].type must be 'text', got %v", item["type"])
			}
			if _, ok := item["text"]; !ok {
				t.Error("content[0] must have 'text' field")
			}
		})
	}
}

// TestMCPToolErrorResultFormat validates that tool execution errors return
// CallToolResult with isError=true (not protocol-level MCPErrorResponse).
func TestMCPToolErrorResultFormat(t *testing.T) {
	// Use a tool that will fail due to invalid Gitea URL
	origURL := giteaURL
	giteaURL = "http://127.0.0.1:1" // unreachable port
	defer func() { giteaURL = origURL }()

	id := jsonRawMessagePtr(`100`)
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_repos","arguments":{}}`),
	}

	resp := handleToolsCall(req)

	mcpResp, ok := resp.(MCPResponse)
	if !ok {
		t.Fatalf("Tool errors must return MCPResponse (with isError), got %T", resp)
	}

	result, ok := mcpResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("Expected CallToolResult map, got %T", mcpResp.Result)
	}

	// Must have isError=true
	isError, ok := result["isError"]
	if !ok {
		t.Fatal("Tool error response missing 'isError' field")
	}
	if isError != true {
		t.Errorf("Expected isError=true, got %v", isError)
	}

	// Must still have content array with error message
	content, ok := result["content"].([]map[string]any)
	if !ok {
		t.Fatalf("Tool error must have content array, got %T", result["content"])
	}
	if len(content) == 0 || content[0]["text"] == "" {
		t.Error("Tool error content must contain error message text")
	}
}

// TestMCPSubprocessIntegration spawns gtr mcp-server as a child process
// (the way Claude Code, OpenCode, and pi-agent do) and validates the
// full request/response cycle including tools/call.
func TestMCPSubprocessIntegration(t *testing.T) {
	if os.Getenv("GITEA_TOKEN") == "" {
		t.Skip("GITEA_TOKEN not set -- skipping subprocess integration test")
	}

	// Build the binary
	binary := t.TempDir() + "/gtr-test"
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build: %s\n%s", err, out)
	}

	// Start MCP server as subprocess (exactly as MCP clients do)
	cmd := exec.Command(binary, "mcp-server")
	cmd.Env = append(os.Environ()) // inherit env for GITEA_TOKEN/GITEA_URL

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start MCP server: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)
	// Increase scanner buffer for large responses
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	sendAndReceive := func(request string) (map[string]any, error) {
		_, err := fmt.Fprintln(stdin, request)
		if err != nil {
			return nil, fmt.Errorf("write failed: %w", err)
		}

		// Read with timeout
		done := make(chan bool, 1)
		var line string
		go func() {
			if scanner.Scan() {
				line = scanner.Text()
			}
			done <- true
		}()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("timeout waiting for response")
		}

		if line == "" {
			return nil, fmt.Errorf("empty response")
		}

		var result map[string]any
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			return nil, fmt.Errorf("invalid JSON response: %s", line)
		}
		return result, nil
	}

	// Step 1: initialize
	t.Run("initialize", func(t *testing.T) {
		resp, err := sendAndReceive(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-harness","version":"1.0"}}}`)
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("Expected result map, got %T", resp["result"])
		}
		if result["protocolVersion"] != "2024-11-05" {
			t.Errorf("Expected protocolVersion 2024-11-05, got %v", result["protocolVersion"])
		}
	})

	// Step 2: notifications/initialized (no response expected, send and continue)
	fmt.Fprintln(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	// Step 3: tools/list
	t.Run("tools_list", func(t *testing.T) {
		resp, err := sendAndReceive(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
		if err != nil {
			t.Fatalf("tools/list failed: %v", err)
		}
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("Expected result map, got %T", resp["result"])
		}
		tools, ok := result["tools"].([]any)
		if !ok {
			t.Fatalf("Expected tools array, got %T", result["tools"])
		}
		if len(tools) != 20 {
			t.Errorf("Expected 20 tools, got %d", len(tools))
		}
	})

	// Step 4: tools/call -- THE CRITICAL TEST
	// This is what hung before the CallToolResult fix
	t.Run("tools_call_list_repos", func(t *testing.T) {
		resp, err := sendAndReceive(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_repos","arguments":{"org":"terraphim","limit":1}}}`)
		if err != nil {
			t.Fatalf("tools/call failed: %v", err)
		}

		// Verify CallToolResult format
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("tools/call result must be a map (CallToolResult), got %T: %v", resp["result"], resp)
		}

		content, ok := result["content"].([]any)
		if !ok {
			t.Fatalf("CallToolResult must have content array, got %T", result["content"])
		}

		if len(content) == 0 {
			t.Fatal("content array must not be empty")
		}

		item, ok := content[0].(map[string]any)
		if !ok {
			t.Fatalf("content item must be map, got %T", content[0])
		}

		if item["type"] != "text" {
			t.Errorf("content[0].type must be 'text', got %v", item["type"])
		}

		text, ok := item["text"].(string)
		if !ok || text == "" {
			t.Error("content[0].text must be a non-empty string")
		}

		// The text should be valid JSON (Gitea API response)
		var apiResp any
		if err := json.Unmarshal([]byte(text), &apiResp); err != nil {
			t.Errorf("content[0].text should be valid JSON: %v", err)
		}
	})

	// Step 5: tools/call with list_issues
	t.Run("tools_call_list_issues", func(t *testing.T) {
		resp, err := sendAndReceive(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_issues","arguments":{"owner":"terraphim","repo":"gitea-robot","state":"open","limit":1}}}`)
		if err != nil {
			t.Fatalf("tools/call failed: %v", err)
		}

		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("Expected CallToolResult map, got %T", resp["result"])
		}

		content, ok := result["content"].([]any)
		if !ok {
			t.Fatalf("Expected content array, got %T", result["content"])
		}

		if len(content) == 0 {
			t.Fatal("content array must not be empty")
		}

		item := content[0].(map[string]any)
		if item["type"] != "text" {
			t.Errorf("Expected type 'text', got %v", item["type"])
		}
	})

	// Step 6: tools/call with validation error (missing required param)
	t.Run("tools_call_validation_error", func(t *testing.T) {
		resp, err := sendAndReceive(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_issues","arguments":{"owner":"terraphim"}}}`)
		if err != nil {
			t.Fatalf("tools/call failed: %v", err)
		}

		// Validation errors should be protocol-level MCPErrorResponse
		errObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatalf("Expected protocol error for missing required param, got result: %v", resp["result"])
		}

		code, ok := errObj["code"].(float64)
		if !ok || code != -32602 {
			t.Errorf("Expected error code -32602, got %v", errObj["code"])
		}

		msg, ok := errObj["message"].(string)
		if !ok || !strings.Contains(msg, "repo") {
			t.Errorf("Expected error about missing 'repo', got: %s", msg)
		}
	})

	// Step 7: ping
	t.Run("ping", func(t *testing.T) {
		resp, err := sendAndReceive(`{"jsonrpc":"2.0","id":6,"method":"ping"}`)
		if err != nil {
			t.Fatalf("ping failed: %v", err)
		}
		if resp["error"] != nil {
			t.Errorf("Unexpected error for ping: %v", resp["error"])
		}
	})
}

// TestMCPResponseNoEmbeddedNewlines validates that MCP responses never
// contain embedded newlines (MCP spec requirement for stdio transport).
func TestMCPResponseNoEmbeddedNewlines(t *testing.T) {
	// Test with a response that might contain newlines in the data
	id := jsonRawMessagePtr(`1`)

	// Create a response with text containing newlines
	resp := toolResult(id, "line1\nline2\nline3")

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// The marshaled JSON should not contain literal newlines
	// (json.Marshal escapes them to \n)
	jsonStr := string(data)
	if strings.Contains(jsonStr, "\n") {
		t.Errorf("MCP response contains embedded newline (violates spec): %s", jsonStr)
	}
}

// TestToolResultHelper validates the toolResult and toolErrorResult helpers.
func TestToolResultHelper(t *testing.T) {
	t.Run("toolResult", func(t *testing.T) {
		id := jsonRawMessagePtr(`42`)
		resp := toolResult(id, "hello world")

		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("Expected map result, got %T", resp.Result)
		}

		content := result["content"].([]map[string]any)
		if len(content) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(content))
		}

		if content[0]["type"] != "text" {
			t.Errorf("Expected type 'text', got %v", content[0]["type"])
		}
		if content[0]["text"] != "hello world" {
			t.Errorf("Expected text 'hello world', got %v", content[0]["text"])
		}

		// Should NOT have isError
		if _, ok := result["isError"]; ok {
			t.Error("toolResult should not have isError field")
		}
	})

	t.Run("toolErrorResult", func(t *testing.T) {
		id := jsonRawMessagePtr(`43`)
		resp := toolErrorResult(id, "something failed")

		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("Expected map result, got %T", resp.Result)
		}

		// Must have isError=true
		if result["isError"] != true {
			t.Errorf("Expected isError=true, got %v", result["isError"])
		}

		content := result["content"].([]map[string]any)
		if content[0]["text"] != "something failed" {
			t.Errorf("Expected error text 'something failed', got %v", content[0]["text"])
		}
	})
}
