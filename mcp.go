package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// MCPRequest represents a JSON-RPC 2.0 request
type MCPRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
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

func mcpServerCmd() {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
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
			sendResponse(writer, resp)
			continue
		}

		var resp any
		switch req.Method {
		case "initialize":
			resp = handleInitialize(req)
		case "notifications/initialized":
			continue
		case "tools/list":
			resp = handleToolsList(req)
		case "tools/call":
			resp = handleToolsCall(req)
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

		sendResponse(writer, resp)
	}
}

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

func captureStdout(fn func()) (string, error) {
	tmpfile, err := os.CreateTemp("", "mcp-tool-*.out")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	old := os.Stdout
	os.Stdout = tmpfile
	fn()
	os.Stdout = old

	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}
	return string(data), nil
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
				{
					"name":        "list_labels",
					"description": "List repository labels",
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
							"limit": map[string]any{
								"type":        "integer",
								"description": "Maximum number of labels to return",
								"default":     50,
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "list_pulls",
					"description": "List pull requests",
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
							"state": map[string]any{
								"type":        "string",
								"description": "PR state: open, closed, or all",
								"default":     "open",
							},
							"labels": map[string]any{
								"type":        "string",
								"description": "Comma-separated label names to filter by",
							},
							"limit": map[string]any{
								"type":        "integer",
								"description": "Maximum number of PRs to return",
								"default":     20,
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "create_pull",
					"description": "Create a pull request",
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
							"title": map[string]any{
								"type":        "string",
								"description": "PR title",
							},
							"head": map[string]any{
								"type":        "string",
								"description": "Source branch",
							},
							"base": map[string]any{
								"type":        "string",
								"description": "Target branch",
								"default":     "main",
							},
							"body": map[string]any{
								"type":        "string",
								"description": "PR body/description",
							},
							"labels": map[string]any{
								"type":        "string",
								"description": "Comma-separated label names",
							},
							"assignees": map[string]any{
								"type":        "string",
								"description": "Comma-separated assignee usernames",
							},
							"draft": map[string]any{
								"type":        "boolean",
								"description": "Create as draft PR",
								"default":     false,
							},
						},
						"required": []string{"owner", "repo", "title", "head"},
					},
				},
				{
					"name":        "merge_pull",
					"description": "Merge a pull request",
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
							"index": map[string]any{
								"type":        "integer",
								"description": "PR number",
							},
							"style": map[string]any{
								"type":        "string",
								"description": "Merge style: merge, rebase, or squash",
								"default":     "merge",
							},
							"title": map[string]any{
								"type":        "string",
								"description": "Merge commit title",
							},
							"message": map[string]any{
								"type":        "string",
								"description": "Merge commit message",
							},
							"delete_branch": map[string]any{
								"type":        "boolean",
								"description": "Delete source branch after merge",
								"default":     false,
							},
						},
						"required": []string{"owner", "repo", "index"},
					},
				},
				{
					"name":        "view_issue",
					"description": "View a single issue with full details",
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
							"index": map[string]any{
								"type":        "integer",
								"description": "Issue number",
							},
						},
						"required": []string{"owner", "repo", "index"},
					},
				},
				{
					"name":        "view_pull",
					"description": "View a single pull request with full details",
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
							"index": map[string]any{
								"type":        "integer",
								"description": "PR number",
							},
						},
						"required": []string{"owner", "repo", "index"},
					},
				},
				{
					"name":        "create_label",
					"description": "Create a repository label",
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
							"name": map[string]any{
								"type":        "string",
								"description": "Label name",
							},
							"colour": map[string]any{
								"type":        "string",
								"description": "Label colour (hex, e.g. #FF0000)",
							},
							"description": map[string]any{
								"type":        "string",
								"description": "Label description",
							},
						},
						"required": []string{"owner", "repo", "name", "colour"},
					},
				},
				{
					"name":        "create_repo",
					"description": "Create a repository",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type":        "string",
								"description": "Repository name",
							},
							"org": map[string]any{
								"type":        "string",
								"description": "Organisation (omit for personal repo)",
							},
							"description": map[string]any{
								"type":        "string",
								"description": "Repository description",
							},
							"private": map[string]any{
								"type":        "boolean",
								"description": "Create as private repository",
								"default":     false,
							},
							"auto_init": map[string]any{
								"type":        "boolean",
								"description": "Initialise with README",
								"default":     false,
							},
							"gitignore": map[string]any{
								"type":        "string",
								"description": "Gitignore template (e.g. Go)",
							},
							"license": map[string]any{
								"type":        "string",
								"description": "License template (e.g. MIT)",
							},
							"default_branch": map[string]any{
								"type":        "string",
								"description": "Default branch name",
								"default":     "main",
							},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "create_release",
					"description": "Create a release",
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
							"tag": map[string]any{
								"type":        "string",
								"description": "Tag name (e.g. v1.0.0)",
							},
							"title": map[string]any{
								"type":        "string",
								"description": "Release title",
							},
							"body": map[string]any{
								"type":        "string",
								"description": "Release body/notes",
							},
							"target": map[string]any{
								"type":        "string",
								"description": "Target branch",
							},
							"draft": map[string]any{
								"type":        "boolean",
								"description": "Create as draft release",
								"default":     false,
							},
							"prerelease": map[string]any{
								"type":        "boolean",
								"description": "Mark as pre-release",
								"default":     false,
							},
						},
						"required": []string{"owner", "repo", "tag"},
					},
				},
				{
					"name":        "list_repos",
					"description": "List repositories",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"org": map[string]any{
								"type":        "string",
								"description": "Organisation name (omit to search all)",
							},
							"query": map[string]any{
								"type":        "string",
								"description": "Search query",
							},
							"limit": map[string]any{
								"type":        "integer",
								"description": "Maximum number of repos to return",
								"default":     20,
							},
						},
					},
				},
				{
					"name":        "fork_repo",
					"description": "Fork a repository",
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
							"org": map[string]any{
								"type":        "string",
								"description": "Fork to organisation (omit for personal fork)",
							},
							"name": map[string]any{
								"type":        "string",
								"description": "Fork name (defaults to original)",
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "list_issues",
					"description": "List repository issues",
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
							"state": map[string]any{
								"type":        "string",
								"description": "Issue state: open, closed, or all",
								"default":     "open",
							},
							"labels": map[string]any{
								"type":        "string",
								"description": "Comma-separated label names to filter by",
							},
							"limit": map[string]any{
								"type":        "integer",
								"description": "Maximum number of issues to return",
								"default":     20,
							},
						},
						"required": []string{"owner", "repo"},
					},
				},
				{
					"name":        "create_issue",
					"description": "Create a new issue",
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
							"title": map[string]any{
								"type":        "string",
								"description": "Issue title",
							},
							"body": map[string]any{
								"type":        "string",
								"description": "Issue body",
							},
							"labels": map[string]any{
								"type":        "string",
								"description": "Comma-separated label names",
							},
						},
						"required": []string{"owner", "repo", "title"},
					},
				},
				{
					"name":        "comment",
					"description": "Add a comment to an issue",
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
								"description": "Issue number",
							},
							"body": map[string]any{
								"type":        "string",
								"description": "Comment body",
							},
						},
						"required": []string{"owner", "repo", "issue", "body"},
					},
				},
				{
					"name":        "close_issue",
					"description": "Close an issue",
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
								"description": "Issue number",
							},
						},
						"required": []string{"owner", "repo", "issue"},
					},
				},
				{
					"name":        "edit_issue",
					"description": "Edit an issue",
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
								"description": "Issue number",
							},
							"title": map[string]any{
								"type":        "string",
								"description": "New issue title",
							},
							"body": map[string]any{
								"type":        "string",
								"description": "New issue body",
							},
							"state": map[string]any{
								"type":        "string",
								"description": "New state: open or closed",
							},
							"add_labels": map[string]any{
								"type":        "string",
								"description": "Comma-separated label names to add",
							},
						},
						"required": []string{"owner", "repo", "issue"},
					},
				},
			},
		},
	}
}

func handleToolsCall(req MCPRequest) any {
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

	switch params.Name {
	case "triage":
		return handleTriageTool(params.Arguments, req.ID)
	case "ready":
		return handleReadyTool(params.Arguments, req.ID)
	case "graph":
		return handleGraphTool(params.Arguments, req.ID)
	case "add_dep":
		return handleAddDepTool(params.Arguments, req.ID)
	case "list_labels":
		return handleListLabelsTool(params.Arguments, req.ID)
	case "list_pulls":
		return handleListPullsTool(params.Arguments, req.ID)
	case "create_pull":
		return handleCreatePullTool(params.Arguments, req.ID)
	case "merge_pull":
		return handleMergePullTool(params.Arguments, req.ID)
	case "view_issue":
		return handleViewIssueTool(params.Arguments, req.ID)
	case "view_pull":
		return handleViewPullTool(params.Arguments, req.ID)
	case "create_label":
		return handleCreateLabelTool(params.Arguments, req.ID)
	case "create_repo":
		return handleCreateRepoTool(params.Arguments, req.ID)
	case "create_release":
		return handleCreateReleaseTool(params.Arguments, req.ID)
	case "list_repos":
		return handleListReposTool(params.Arguments, req.ID)
	case "fork_repo":
		return handleForkRepoTool(params.Arguments, req.ID)
	case "list_issues":
		return handleListIssuesTool(params.Arguments, req.ID)
	case "create_issue":
		return handleCreateIssueTool(params.Arguments, req.ID)
	case "comment":
		return handleCommentTool(params.Arguments, req.ID)
	case "close_issue":
		return handleCloseIssueTool(params.Arguments, req.ID)
	case "edit_issue":
		return handleEditIssueTool(params.Arguments, req.ID)
	default:
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Tool not found: " + params.Name,
			},
		}
	}
}

func handleTriageTool(args json.RawMessage, id *json.RawMessage) any {
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
				Code:    -32602,
				Message: "Invalid arguments for triage: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: repo",
			},
		}
	}

	format := "json"
	if argsStruct.Format != nil && *argsStruct.Format != "" {
		format = *argsStruct.Format
	}
	_ = format

	url := fmt.Sprintf("%s/api/v1/robot/triage?owner=%s&repo=%s", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output := apiGet(url)

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func handleReadyTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments for ready: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: repo",
			},
		}
	}

	url := fmt.Sprintf("%s/api/v1/robot/ready?owner=%s&repo=%s", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output := apiGet(url)

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func handleGraphTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments for graph: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: repo",
			},
		}
	}

	url := fmt.Sprintf("%s/api/v1/robot/graph?owner=%s&repo=%s", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output := apiGet(url)

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func handleAddDepTool(args json.RawMessage, id *json.RawMessage) any {
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
				Code:    -32602,
				Message: "Invalid arguments for add_dep: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: owner",
			},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: repo",
			},
		}
	}
	if argsStruct.Issue == nil || *argsStruct.Issue == 0 {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: issue",
			},
		}
	}

	if argsStruct.Blocks == nil && argsStruct.RelatesTo == nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing required argument: either blocks or relates_to must be provided",
			},
		}
	}

	dependsOnIndex := int64(0)
	if argsStruct.Blocks != nil {
		dependsOnIndex = *argsStruct.Blocks
	} else if argsStruct.RelatesTo != nil {
		dependsOnIndex = *argsStruct.RelatesTo
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/dependencies", giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Issue)
	body := fmt.Sprintf(`{"index": %d, "owner": %q, "repo": %q}`, dependsOnIndex, *argsStruct.Owner, *argsStruct.Repo)

	output, err := apiPostSafe(url, body)
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func handlePing(req MCPRequest) any {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]string{},
	}
}

func handleListLabelsTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
		Limit *int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments for list_labels: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: owner"},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: repo"},
		}
	}

	limit := 50
	if argsStruct.Limit != nil && *argsStruct.Limit > 0 {
		limit = *argsStruct.Limit
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=%d", giteaURL, *argsStruct.Owner, *argsStruct.Repo, limit)
	data, err := apiGetSafe(url)
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32603, Message: err.Error()},
		}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: data}
}

func handleListPullsTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner  *string `json:"owner,omitempty"`
		Repo   *string `json:"repo,omitempty"`
		State  *string `json:"state,omitempty"`
		Labels *string `json:"labels,omitempty"`
		Limit  *int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments for list_pulls: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: owner"},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: repo"},
		}
	}

	state := "open"
	if argsStruct.State != nil && *argsStruct.State != "" {
		state = *argsStruct.State
	}
	limit := 20
	if argsStruct.Limit != nil && *argsStruct.Limit > 0 {
		limit = *argsStruct.Limit
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=%s&limit=%d",
		giteaURL, *argsStruct.Owner, *argsStruct.Repo, state, limit)
	if argsStruct.Labels != nil && *argsStruct.Labels != "" {
		url += "&labels=" + *argsStruct.Labels
	}

	data, err := apiGetSafe(url)
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32603, Message: err.Error()},
		}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: data}
}

func handleCreatePullTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner     *string `json:"owner,omitempty"`
		Repo      *string `json:"repo,omitempty"`
		Title     *string `json:"title,omitempty"`
		Head      *string `json:"head,omitempty"`
		Base      *string `json:"base,omitempty"`
		Body      *string `json:"body,omitempty"`
		Labels    *string `json:"labels,omitempty"`
		Assignees *string `json:"assignees,omitempty"`
		Draft     *bool   `json:"draft,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments for create_pull: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: owner"},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: repo"},
		}
	}
	if argsStruct.Title == nil || *argsStruct.Title == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: title"},
		}
	}
	if argsStruct.Head == nil || *argsStruct.Head == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: head"},
		}
	}

	base := "main"
	if argsStruct.Base != nil && *argsStruct.Base != "" {
		base = *argsStruct.Base
	}

	payload := map[string]any{
		"title": *argsStruct.Title,
		"head":  *argsStruct.Head,
		"base":  base,
	}

	if argsStruct.Body != nil {
		payload["body"] = *argsStruct.Body
	}
	if argsStruct.Draft != nil && *argsStruct.Draft {
		payload["draft"] = true
	}

	if argsStruct.Labels != nil && *argsStruct.Labels != "" {
		names := strings.Split(*argsStruct.Labels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err := resolveLabels(*argsStruct.Owner, *argsStruct.Repo, names)
		if err == nil && len(labelIDs) > 0 {
			payload["labels"] = labelIDs
		}
	}

	if argsStruct.Assignees != nil && *argsStruct.Assignees != "" {
		names := strings.Split(*argsStruct.Assignees, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		payload["assignees"] = names
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32603, Message: err.Error()},
		}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleMergePullTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner        *string `json:"owner,omitempty"`
		Repo         *string `json:"repo,omitempty"`
		Index        *int64  `json:"index,omitempty"`
		Style        *string `json:"style,omitempty"`
		Title        *string `json:"title,omitempty"`
		Message      *string `json:"message,omitempty"`
		DeleteBranch *bool   `json:"delete_branch,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid arguments for merge_pull: " + err.Error(),
			},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: owner"},
		}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: repo"},
		}
	}
	if argsStruct.Index == nil || *argsStruct.Index == 0 {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Missing required argument: index"},
		}
	}

	style := "merge"
	if argsStruct.Style != nil && *argsStruct.Style != "" {
		style = *argsStruct.Style
	}

	payload := map[string]any{
		"Do": style,
	}
	if argsStruct.DeleteBranch != nil && *argsStruct.DeleteBranch {
		payload["delete_branch_after_merge"] = true
	}
	if argsStruct.Title != nil && *argsStruct.Title != "" {
		payload["merge_title_field"] = *argsStruct.Title
	}
	if argsStruct.Message != nil && *argsStruct.Message != "" {
		payload["merge_message_field"] = *argsStruct.Message
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge",
		giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Index)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32603, Message: err.Error()},
		}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleViewIssueTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
		Index *int64  `json:"index,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Invalid arguments for view_issue: " + err.Error()},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Index == nil || *argsStruct.Index == 0 {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: index"}}
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Index)
	data, err := apiGetSafe(url)
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: data}
}

func handleViewPullTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
		Index *int64  `json:"index,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Invalid arguments for view_pull: " + err.Error()},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Index == nil || *argsStruct.Index == 0 {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: index"}}
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d", giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Index)
	data, err := apiGetSafe(url)
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: data}
}

func handleCreateLabelTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner       *string `json:"owner,omitempty"`
		Repo        *string `json:"repo,omitempty"`
		Name        *string `json:"name,omitempty"`
		Colour      *string `json:"colour,omitempty"`
		Description *string `json:"description,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Invalid arguments for create_label: " + err.Error()},
		}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Name == nil || *argsStruct.Name == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: name"}}
	}
	if argsStruct.Colour == nil || *argsStruct.Colour == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: colour"}}
	}

	c := *argsStruct.Colour
	if len(c) > 0 && c[0] != '#' {
		c = "#" + c
	}

	payload := map[string]any{
		"name":  *argsStruct.Name,
		"color": c,
	}
	if argsStruct.Description != nil && *argsStruct.Description != "" {
		payload["description"] = *argsStruct.Description
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleCreateRepoTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Name          *string `json:"name,omitempty"`
		Org           *string `json:"org,omitempty"`
		Description   *string `json:"description,omitempty"`
		Private       *bool   `json:"private,omitempty"`
		AutoInit      *bool   `json:"auto_init,omitempty"`
		Gitignore     *string `json:"gitignore,omitempty"`
		License       *string `json:"license,omitempty"`
		DefaultBranch *string `json:"default_branch,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "Invalid arguments for create_repo: " + err.Error()},
		}
	}

	if argsStruct.Name == nil || *argsStruct.Name == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: name"}}
	}

	payload := map[string]any{
		"name":           *argsStruct.Name,
		"private":        argsStruct.Private != nil && *argsStruct.Private,
		"auto_init":      argsStruct.AutoInit != nil && *argsStruct.AutoInit,
		"default_branch": "main",
	}
	if argsStruct.DefaultBranch != nil && *argsStruct.DefaultBranch != "" {
		payload["default_branch"] = *argsStruct.DefaultBranch
	}
	if argsStruct.Description != nil && *argsStruct.Description != "" {
		payload["description"] = *argsStruct.Description
	}
	if argsStruct.Gitignore != nil && *argsStruct.Gitignore != "" {
		payload["gitignores"] = *argsStruct.Gitignore
	}
	if argsStruct.License != nil && *argsStruct.License != "" {
		payload["license"] = *argsStruct.License
	}

	var url string
	if argsStruct.Org != nil && *argsStruct.Org != "" {
		url = fmt.Sprintf("%s/api/v1/orgs/%s/repos", giteaURL, *argsStruct.Org)
	} else {
		url = fmt.Sprintf("%s/api/v1/user/repos", giteaURL)
	}

	jsonBody, _ := json.Marshal(payload)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleCreateReleaseTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner      *string `json:"owner,omitempty"`
		Repo       *string `json:"repo,omitempty"`
		Tag        *string `json:"tag,omitempty"`
		Title      *string `json:"title,omitempty"`
		Body       *string `json:"body,omitempty"`
		Target     *string `json:"target,omitempty"`
		Draft      *bool   `json:"draft,omitempty"`
		Prerelease *bool   `json:"prerelease,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for create_release: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Tag == nil || *argsStruct.Tag == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: tag"}}
	}

	payload := map[string]any{
		"tag_name":   *argsStruct.Tag,
		"draft":      argsStruct.Draft != nil && *argsStruct.Draft,
		"prerelease": argsStruct.Prerelease != nil && *argsStruct.Prerelease,
	}
	if argsStruct.Title != nil && *argsStruct.Title != "" {
		payload["name"] = *argsStruct.Title
	} else {
		payload["name"] = *argsStruct.Tag
	}
	if argsStruct.Body != nil && *argsStruct.Body != "" {
		payload["body"] = *argsStruct.Body
	}
	if argsStruct.Target != nil && *argsStruct.Target != "" {
		payload["target_commitish"] = *argsStruct.Target
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleListReposTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Org   *string `json:"org,omitempty"`
		Limit *int    `json:"limit,omitempty"`
		Query *string `json:"query,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for list_repos: " + err.Error()}}
	}

	limit := 20
	if argsStruct.Limit != nil && *argsStruct.Limit > 0 {
		limit = *argsStruct.Limit
	}

	var url string
	if argsStruct.Org != nil && *argsStruct.Org != "" {
		url = fmt.Sprintf("%s/api/v1/orgs/%s/repos?limit=%d", giteaURL, *argsStruct.Org, limit)
	} else if argsStruct.Query != nil && *argsStruct.Query != "" {
		url = fmt.Sprintf("%s/api/v1/repos/search?q=%s&limit=%d", giteaURL, *argsStruct.Query, limit)
	} else {
		url = fmt.Sprintf("%s/api/v1/repos/search?limit=%d", giteaURL, limit)
	}

	data, err := apiGetSafe(url)
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: data}
}

func handleForkRepoTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
		Org   *string `json:"org,omitempty"`
		Name  *string `json:"name,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for fork_repo: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}

	payload := map[string]any{}
	if argsStruct.Org != nil && *argsStruct.Org != "" {
		payload["organization"] = *argsStruct.Org
	}
	if argsStruct.Name != nil && *argsStruct.Name != "" {
		payload["name"] = *argsStruct.Name
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/forks", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleListIssuesTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner  *string `json:"owner,omitempty"`
		Repo   *string `json:"repo,omitempty"`
		State  *string `json:"state,omitempty"`
		Labels *string `json:"labels,omitempty"`
		Limit  *int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for list_issues: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}

	state := "open"
	if argsStruct.State != nil && *argsStruct.State != "" {
		state = *argsStruct.State
	}
	limit := 20
	if argsStruct.Limit != nil && *argsStruct.Limit > 0 {
		limit = *argsStruct.Limit
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=%s&limit=%d&type=issues",
		giteaURL, *argsStruct.Owner, *argsStruct.Repo, state, limit)
	if argsStruct.Labels != nil && *argsStruct.Labels != "" {
		url += "&labels=" + *argsStruct.Labels
	}

	data, err := apiGetSafe(url)
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: data}
}

func handleCreateIssueTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner  *string `json:"owner,omitempty"`
		Repo   *string `json:"repo,omitempty"`
		Title  *string `json:"title,omitempty"`
		Body   *string `json:"body,omitempty"`
		Labels *string `json:"labels,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for create_issue: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Title == nil || *argsStruct.Title == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: title"}}
	}

	payload := map[string]any{
		"title": *argsStruct.Title,
	}
	if argsStruct.Body != nil {
		payload["body"] = *argsStruct.Body
	}

	if argsStruct.Labels != nil && *argsStruct.Labels != "" {
		names := strings.Split(*argsStruct.Labels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err := resolveLabels(*argsStruct.Owner, *argsStruct.Repo, names)
		if err == nil && len(labelIDs) > 0 {
			payload["labels"] = labelIDs
		}
	}

	jsonBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues", giteaURL, *argsStruct.Owner, *argsStruct.Repo)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleCommentTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
		Issue *int64  `json:"issue,omitempty"`
		Body  *string `json:"body,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for comment: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Issue == nil || *argsStruct.Issue == 0 {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: issue"}}
	}
	if argsStruct.Body == nil || *argsStruct.Body == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: body"}}
	}

	jsonBody, _ := json.Marshal(map[string]string{"body": *argsStruct.Body})
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments",
		giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Issue)
	output, err := apiPostSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleCloseIssueTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner *string `json:"owner,omitempty"`
		Repo  *string `json:"repo,omitempty"`
		Issue *int64  `json:"issue,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for close_issue: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Issue == nil || *argsStruct.Issue == 0 {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: issue"}}
	}

	jsonBody, _ := json.Marshal(map[string]string{"state": "closed"})
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d",
		giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Issue)
	output, err := apiPatchSafe(url, string(jsonBody))
	if err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: output}
}

func handleEditIssueTool(args json.RawMessage, id *json.RawMessage) any {
	var argsStruct struct {
		Owner     *string `json:"owner,omitempty"`
		Repo      *string `json:"repo,omitempty"`
		Issue     *int64  `json:"issue,omitempty"`
		Title     *string `json:"title,omitempty"`
		Body      *string `json:"body,omitempty"`
		State     *string `json:"state,omitempty"`
		AddLabels *string `json:"add_labels,omitempty"`
	}
	if err := json.Unmarshal(args, &argsStruct); err != nil {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Invalid arguments for edit_issue: " + err.Error()}}
	}

	if argsStruct.Owner == nil || *argsStruct.Owner == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: owner"}}
	}
	if argsStruct.Repo == nil || *argsStruct.Repo == "" {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: repo"}}
	}
	if argsStruct.Issue == nil || *argsStruct.Issue == 0 {
		return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32602, Message: "Missing required argument: issue"}}
	}

	payload := map[string]any{}
	if argsStruct.Title != nil && *argsStruct.Title != "" {
		payload["title"] = *argsStruct.Title
	}
	if argsStruct.Body != nil {
		payload["body"] = *argsStruct.Body
	}
	if argsStruct.State != nil && *argsStruct.State != "" {
		payload["state"] = *argsStruct.State
	}

	var result string
	if len(payload) > 0 {
		jsonBody, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d",
			giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Issue)
		output, err := apiPatchSafe(url, string(jsonBody))
		if err != nil {
			return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
		}
		result = output
	}

	if argsStruct.AddLabels != nil && *argsStruct.AddLabels != "" {
		names := strings.Split(*argsStruct.AddLabels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err := resolveLabels(*argsStruct.Owner, *argsStruct.Repo, names)
		if err == nil && len(labelIDs) > 0 {
			jsonBody, _ := json.Marshal(map[string]any{"labels": labelIDs})
			url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels",
				giteaURL, *argsStruct.Owner, *argsStruct.Repo, *argsStruct.Issue)
			labelResult, err := apiPostSafe(url, string(jsonBody))
			if err != nil {
				return MCPErrorResponse{JSONRPC: "2.0", ID: id, Error: &MCPError{Code: -32603, Message: err.Error()}}
			}
			if result == "" {
				result = labelResult
			}
		}
	}

	if result == "" {
		result = fmt.Sprintf(`{"message":"no changes applied to issue #%d"}`, *argsStruct.Issue)
	}

	return MCPResponse{JSONRPC: "2.0", ID: id, Result: result}
}
