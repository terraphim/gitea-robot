// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// gitea-robot CLI - thin wrapper for Gitea Robot API
// Usage: go run cmd/gitea-robot/main.go [command] [flags]

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	giteaURL   = os.Getenv("GITEA_URL")
	giteaToken = os.Getenv("GITEA_TOKEN")
)

func main() {
	// Set default URL
	if giteaURL == "" {
		giteaURL = "http://localhost:3000"
	}

	// Handle help flags before checking for GITEA_TOKEN
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}

	// Check for GITEA_TOKEN after help check
	if giteaToken == "" {
		fmt.Fprintln(os.Stderr, "Error: GITEA_TOKEN environment variable required")
		os.Exit(1)
	}

	command := os.Args[1]
	os.Args = os.Args[1:] // Remove command from args

	switch command {
	case "triage":
		triageCmd()
	case "ready":
		readyCmd()
	case "graph":
		graphCmd()
	case "add-dep":
		addDepCmd()
	case "list-issues":
		listIssuesCmd()
	case "create-issue":
		createIssueCmd()
	case "comment":
		commentCmd()
	case "close-issue":
		closeIssueCmd()
	case "edit-issue":
		editIssueCmd()
	case "mcp-server":
		mcpServerCmd()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gitea-robot - CLI for Gitea Robot API

Usage:
  gitea-robot [command] [flags]

Commands:
  triage      Get prioritized task list
  ready       Get unblocked (ready) tasks
  graph       Get dependency graph
  add-dep     Add dependency between issues
  list-issues   List repository issues (filtered by state, labels)
  create-issue  Create a new issue
  comment       Add a comment to an issue
  close-issue   Close an issue
  edit-issue    Edit an issue (title, state, labels)
  mcp-server    Start MCP server exposing gitea-robot functionality

Environment:
  GITEA_URL    Gitea instance URL (default: http://localhost:3000)
  GITEA_TOKEN  API token for authentication

Examples:
  # Get triage report
  gitea-robot triage --owner terraphim --repo gitea

  # Get ready issues
  gitea-robot ready --owner terraphim --repo gitea

  # Add dependency: issue 2 blocked by issue 1
  gitea-robot add-dep --owner terraphim --repo gitea --issue 2 --blocks 1

  # List open issues
  gitea-robot list-issues --owner terraphim --repo terraphim-ai

  # Create an issue with labels
  gitea-robot create-issue --owner terraphim --repo terraphim-ai --title "Fix bug" --labels "priority/P1-high"

  # Comment on an issue (body from file)
  gitea-robot comment --owner terraphim --repo terraphim-ai --issue 42 --body-file report.md

  # Close an issue
  gitea-robot close-issue --owner terraphim --repo terraphim-ai --issue 42

  # Start MCP server
  gitea-robot mcp-server`)
}

func triageCmd() {
	fs := flag.NewFlagSet("triage", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	format := fs.String("format", "json", "Output format: json or markdown")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/robot/triage?owner=%s&repo=%s", giteaURL, *owner, *repo)
	data := apiGet(url)

	if *format == "json" {
		fmt.Println(data)
	} else {
		// Pretty print as markdown
		var result map[string]any
		json.Unmarshal([]byte(data), &result)
		printTriageMarkdown(result)
	}
}

func readyCmd() {
	fs := flag.NewFlagSet("ready", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/robot/ready?owner=%s&repo=%s", giteaURL, *owner, *repo)
	data := apiGet(url)
	fmt.Println(data)
}

func graphCmd() {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/robot/graph?owner=%s&repo=%s", giteaURL, *owner, *repo)
	data := apiGet(url)
	fmt.Println(data)
}

func addDepCmd() {
	fs := flag.NewFlagSet("add-dep", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue index (the one being blocked)")
	blocks := fs.Int64("blocks", 0, "Issue index that blocks this issue")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	dependsOn := *blocks
	if dependsOn == 0 {
		fmt.Fprintln(os.Stderr, "Error: --blocks required")
		os.Exit(1)
	}

	// Gitea dependency API uses IssueMeta: index + owner + repo (not internal ID)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/dependencies", giteaURL, *owner, *repo, *issue)
	body := fmt.Sprintf(`{"index": %d, "owner": %q, "repo": %q}`, dependsOn, *owner, *repo)

	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Authorization", "token "+giteaToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Dependency added successfully")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}
}

func listIssuesCmd() {
	fs := flag.NewFlagSet("list-issues", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	state := fs.String("state", "open", "Issue state: open, closed, or all")
	labels := fs.String("labels", "", "Comma-separated label names to filter by")
	limit := fs.Int("limit", 20, "Maximum number of issues to return")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=%s&limit=%d&type=issues",
		giteaURL, *owner, *repo, *state, *limit)
	if *labels != "" {
		u += "&labels=" + *labels
	}
	data := apiGet(u)
	fmt.Println(data)
}

func createIssueCmd() {
	fs := flag.NewFlagSet("create-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	title := fs.String("title", "", "Issue title")
	body := fs.String("body", "", "Issue body")
	bodyFile := fs.String("body-file", "", "Read issue body from file")
	labels := fs.String("labels", "", "Comma-separated label names")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --title required")
		fs.Usage()
		os.Exit(1)
	}

	issueBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}

	// Resolve label names to IDs if provided
	var labelIDs []int64
	if *labels != "" {
		names := strings.Split(*labels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err = resolveLabels(*owner, *repo, names)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not resolve labels: %v\n", err)
		}
	}

	payload := map[string]any{
		"title": *title,
		"body":  issueBody,
	}
	if len(labelIDs) > 0 {
		payload["labels"] = labelIDs
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var issue map[string]any
	if err := json.Unmarshal([]byte(result), &issue); err == nil {
		if num, ok := issue["number"].(float64); ok {
			fmt.Printf("Created issue #%.0f: %s\n", num, *title)
			return
		}
	}
	fmt.Println(result)
}

func commentCmd() {
	fs := flag.NewFlagSet("comment", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue number")
	body := fs.String("body", "", "Comment body")
	bodyFile := fs.String("body-file", "", "Read comment body from file")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	commentBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}
	if commentBody == "" {
		fmt.Fprintln(os.Stderr, "Error: --body or --body-file required")
		os.Exit(1)
	}

	payload, _ := json.Marshal(map[string]string{"body": commentBody})
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", giteaURL, *owner, *repo, *issue)
	_, err = apiPostSafe(u, string(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Comment added to issue #%d\n", *issue)
}

func closeIssueCmd() {
	fs := flag.NewFlagSet("close-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue number")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	payload, _ := json.Marshal(map[string]string{"state": "closed"})
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", giteaURL, *owner, *repo, *issue)
	_, err := apiPatchSafe(u, string(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Issue #%d closed\n", *issue)
}

func editIssueCmd() {
	fs := flag.NewFlagSet("edit-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue number")
	title := fs.String("title", "", "New issue title")
	body := fs.String("body", "", "New issue body")
	bodyFile := fs.String("body-file", "", "Read issue body from file")
	state := fs.String("state", "", "New state: open or closed")
	addLabels := fs.String("add-labels", "", "Comma-separated label names to add")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	payload := map[string]any{}
	if *title != "" {
		payload["title"] = *title
	}
	if *state != "" {
		payload["state"] = *state
	}
	issueBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}
	if issueBody != "" {
		payload["body"] = issueBody
	}

	if len(payload) > 0 {
		jsonBody, _ := json.Marshal(payload)
		u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", giteaURL, *owner, *repo, *issue)
		_, err := apiPatchSafe(u, string(jsonBody))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating issue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Issue #%d updated\n", *issue)
	}

	if *addLabels != "" {
		names := strings.Split(*addLabels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err := resolveLabels(*owner, *repo, names)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving labels: %v\n", err)
			os.Exit(1)
		}
		if len(labelIDs) > 0 {
			jsonBody, _ := json.Marshal(map[string]any{"labels": labelIDs})
			u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", giteaURL, *owner, *repo, *issue)
			_, err := apiPostSafe(u, string(jsonBody))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding labels: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Labels added to issue #%d\n", *issue)
		}
	}
}

func apiGet(url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", "token "+giteaToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	return string(body)
}

func apiGetSafe(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "token "+giteaToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %s\n%s", resp.Status, string(body))
	}

	return string(body), nil
}

func apiPatchSafe(url, body string) (string, error) {
	req, err := http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "token "+giteaToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error: %s\n%s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}

func resolveLabels(owner, repo string, names []string) ([]int64, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=50", giteaURL, owner, repo)
	data, err := apiGetSafe(url)
	if err != nil {
		return nil, err
	}

	var labels []map[string]any
	if err := json.Unmarshal([]byte(data), &labels); err != nil {
		return nil, fmt.Errorf("error parsing labels: %v", err)
	}

	nameToID := make(map[string]int64)
	for _, l := range labels {
		name, _ := l["name"].(string)
		id, _ := l["id"].(float64)
		nameToID[name] = int64(id)
	}

	var ids []int64
	for _, name := range names {
		if id, ok := nameToID[name]; ok {
			ids = append(ids, id)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: label %q not found\n", name)
		}
	}
	return ids, nil
}

func readBody(bodyFlag, bodyFileFlag string) (string, error) {
	if bodyFileFlag != "" {
		data, err := os.ReadFile(bodyFileFlag)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return bodyFlag, nil
}

func printTriageMarkdown(result map[string]any) {
	fmt.Println("## Triage Report")
	fmt.Println()

	if quickRef, ok := result["quick_ref"].(map[string]any); ok {
		fmt.Printf("**Stats:** Total: %.0f, Open: %.0f, Blocked: %.0f, Ready: %.0f\n\n",
			quickRef["total"], quickRef["open"], quickRef["blocked"], quickRef["ready"])
	}

	if recs, ok := result["recommendations"].([]any); ok {
		fmt.Println("### Top Recommendations")
		for i, r := range recs {
			if i >= 5 {
				break
			}
			rec := r.(map[string]any)
			fmt.Printf("%d. **#%.0f: %s** (PageRank: %.4f)\n",
				i+1, rec["index"], rec["title"], rec["pagerank"])
		}
	}
}

// captureStdout captures the stdout of the given function and returns it as a string.
// It temporarily redirects os.Stdout to a temporary file, executes the function,
// restores os.Stdout, reads the temporary file, and returns its contents.
// Note: stderr is not captured and will go to the actual stderr of the process.
func captureStdout(fn func()) (string, error) {
	tmpfile, err := os.CreateTemp("", "mcp-tool-*.out")
	if err != nil {
		return "", err
	}
	// Ensure the temporary file is removed when done.
	defer os.Remove(tmpfile.Name())

	old := os.Stdout
	os.Stdout = tmpfile
	fn()
	os.Stdout = old

	// Close the file to flush content.
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// mcpServerCmd implements the MCP server functionality
func mcpServerCmd() {
	// Create buffered reader and writer for stdio communication
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	// Process MCP messages in a loop
	for {
		// Read a line from stdin (MCP messages are newline-delimited JSON-RPC 2.0)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// End of input, exit gracefully
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}

		// Trim whitespace (including newline)
		line = strings.TrimSpace(line)
		if line == "" {
			// Skip empty lines
			continue
		}

		// Parse the JSON-RPC 2.0 request
		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// Send parse error response
			resp := MCPErrorResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &MCPError{
					Code:    -32703, // Parse error
					Message: "Failed to parse JSON: " + err.Error(),
				},
			}
			sendResponse(writer, resp)
			continue
		}

		// Handle the request based on method
		var resp any
		switch req.Method {
		case "initialize":
			resp = handleInitialize(req)
		case "notifications/initialized":
			// This is a notification, no response needed
			continue
		case "tools/list":
			resp = handleToolsList(req)
		case "tools/call":
			resp = handleToolsCall(req)
		case "ping":
			resp = handlePing(req)
		default:
			// Method not found
			resp = MCPErrorResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &MCPError{
					Code:    -32601, // Method not found
					Message: "Method not found: " + req.Method,
				},
			}
		}

		// Send the response
		sendResponse(writer, resp)
	}
}

// sendResponse writes a JSON-RPC response to the writer
func sendResponse(writer *bufio.Writer, resp any) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
		os.Exit(1)
	}
	_, err = writer.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
		os.Exit(1)
	}
	err = writer.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing writer: %v\n", err)
		os.Exit(1)
	}
}

// MCPRequest represents a JSON-RPC 2.0 request
type MCPRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"` // Can be string, number, or null
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

// MCPResponse represents a successful JSON-RPC 2.0 response
type MCPResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
}

// MCPErrorResponse represents an error JSON-RPC 2.0 response
type MCPErrorResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Error   *MCPError        `json:"error,omitempty"`
}

// MCPError represents an error in JSON-RPC 2.0
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// handleInitialize handles the initialize request
func handleInitialize(req MCPRequest) any {
	// Parse the protocol version from the request
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

// handleToolsList returns the list of available tools
func handleToolsList(req MCPRequest) any {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": []map[string]any{
				{
					"name":        "triage",
					"description": "Get prioritized task list with PageRank scores",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"owner": map[string]any{
								"type":        "string",
								"description": "Repository owner",
							},
							"repo": map[string]any{
								"type":        "string",
								"description": "Repository name",
							},
							"format": map[string]any{
								"type":        "string",
								"description": "Output format: json or markdown",
								"default":     "json",
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "ready",
					"description": "Get unblocked (ready) tasks",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"owner": map[string]any{
								"type":        "string",
								"description": "Repository owner",
							},
							"repo": map[string]any{
								"type":        "string",
								"description": "Repository name",
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "graph",
					"description": "Get dependency graph",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"owner": map[string]any{
								"type":        "string",
								"description": "Repository owner",
							},
							"repo": map[string]any{
								"type":        "string",
								"description": "Repository name",
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "add_dep",
					"description": "Add dependency between issues",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"owner": map[string]any{
								"type":        "string",
								"description": "Repository owner",
							},
							"repo": map[string]any{
								"type":        "string",
								"description": "Repository name",
							},
							"issue": map[string]any{
								"type":        "integer",
								"description": "Issue ID (the one being blocked)",
							},
							"blocks": map[string]any{
								"type":        "integer",
								"description": "Issue ID that blocks this issue",
							},
							"relates_to": map[string]any{
								"type":        "integer",
								"description": "Issue ID that relates to this issue",
							},
						},
						"required": []string{"owner", "repo", "issue"},
					},
				},
			},
		},
	}
}

// handleToolsCall handles tool execution requests
func handleToolsCall(req MCPRequest) any {
	// Parse the params to get tool name and arguments
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Invalid arguments: " + err.Error(),
			},
		}
	}

	// Handle the tool call based on name
	switch params.Name {
	case "triage":
		return handleTriageTool(params.Arguments, req.ID)
	case "ready":
		return handleReadyTool(params.Arguments, req.ID)
	case "graph":
		return handleGraphTool(params.Arguments, req.ID)
	case "add_dep":
		return handleAddDepTool(params.Arguments, req.ID)
	default:
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601, // Method not found
				Message: "Tool not found: " + params.Name,
			},
		}
	}
}

// handleTriageTool executes the triage tool
func handleTriageTool(args json.RawMessage, id *json.RawMessage) any {
	// Parse arguments
	var argsStruct struct {
		Owner  *string `json:"owner,omitempty"`
		Repo   *string `json:"repo,omitempty"`
		Format *string `json:"format,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Invalid arguments for triage: " + err.Error(),
			},
		}
	}

	// Validate required arguments
	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: repo",
			},
		}
	}

	// Set format with default
	format := "json"
	if argsStruct.Format != nil && *argsStruct.Format != "" {
		format = *argsStruct.Format
	}

	// Call API directly instead of using triageCmd to avoid os.Exit()
	url := fmt.Sprintf("%s/api/v1/robot/triage?owner=%s&repo=%s", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output := apiGet(url)

	// For markdown format, we would need to parse and format the JSON
	// For now, return JSON regardless of format parameter
	_ = format

	// Return the output as the result
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// handleReadyTool executes the ready tool
func handleReadyTool(args json.RawMessage, id *json.RawMessage) any {
	// Parse arguments
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Invalid arguments for ready: " + err.Error(),
			},
		}
	}

	// Validate required arguments
	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: repo",
			},
		}
	}

	// Call API directly instead of using readyCmd to avoid os.Exit()
	url := fmt.Sprintf("%s/api/v1/robot/ready?owner=%s&repo=%s", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output := apiGet(url)

	// Return the output as the result
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// handleGraphTool executes the graph tool
func handleGraphTool(args json.RawMessage, id *json.RawMessage) any {
	// Parse arguments
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Invalid arguments for graph: " + err.Error(),
			},
		}
	}

	// Validate required arguments
	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: repo",
			},
		}
	}

	// Call API directly instead of using graphCmd to avoid os.Exit()
	url := fmt.Sprintf("%s/api/v1/robot/graph?owner=%s&repo=%s", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output := apiGet(url)

	// Return the output as the result
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// handleAddDepTool executes the add-dep tool
func handleAddDepTool(args json.RawMessage, id *json.RawMessage) any {
	// Parse arguments
	var argsStruct struct {
		Owner     *string `json:"owner,omitempty"`
		Repo      *string `json:"repo,omitempty"`
		Issue     *int64  `json:"issue,omitempty"`
		Blocks    *int64  `json:"blocks,omitempty"`
		RelatesTo *int64  `json:"relates_to,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Invalid arguments for add_dep: " + err.Error(),
			},
		}
	}

	// Validate required arguments
	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: repo",
			},
		}
	}
	if argsStruct.Issue == nil || *argsStruct.Issue == 0 {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: issue",
			},
		}
	}

	// Validate that either blocks or relates_to is provided
	if argsStruct.Blocks == nil && argsStruct.RelatesTo == nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602, // Invalid params
				Message: "Missing required argument: either blocks or relates_to must be provided",
			},
		}
	}

	// Determine the dependency issue index
	dependsOnIndex := int64(0)
	if argsStruct.Blocks != nil {
		dependsOnIndex = *argsStruct.Blocks
	} else if argsStruct.RelatesTo != nil {
		dependsOnIndex = *argsStruct.RelatesTo
	}

	// Gitea dependency API uses IssueMeta: index + owner + repo
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/dependencies", giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Issue)
	body := fmt.Sprintf(`{"index": %d, "owner": %q, "repo": %q}`, dependsOnIndex, *argsStruct.Owner, *argsStruct.Repo)

	output, err := apiPostSafe(url, body)
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603, // Internal error
				Message: err.Error(),
			},
		}
	}

	// Return the output as the result
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// apiPostSafe performs a POST request and returns error instead of calling os.Exit
func apiPostSafe(url, body string) (string, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "token "+giteaToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error: %s\n%s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}

// handlePing handles ping requests
func handlePing(req MCPRequest) any {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]string{},
	}
}
