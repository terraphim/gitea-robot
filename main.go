// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// gitea-robot CLI - thin wrapper for Gitea Robot API

package main

import (
	"fmt"
	"os"
)

var (
	giteaURL   = os.Getenv("GITEA_URL")
	giteaToken = os.Getenv("GITEA_TOKEN")
)

func main() {
	if giteaURL == "" {
		giteaURL = "http://localhost:3000"
	}

	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}

	if giteaToken == "" {
		fmt.Fprintln(os.Stderr, "Error: GITEA_TOKEN environment variable required")
		os.Exit(1)
	}

	command := os.Args[1]
	os.Args = os.Args[1:]

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
	case "list-labels":
		listLabelsCmd()
	case "list-pulls":
		listPullsCmd()
	case "create-pull":
		createPullCmd()
	case "merge-pull":
		mergePullCmd()
	case "view-issue":
		viewIssueCmd()
	case "view-pull":
		viewPullCmd()
	case "create-label":
		createLabelCmd()
	case "create-repo":
		createRepoCmd()
	case "create-release":
		createReleaseCmd()
	case "list-repos":
		listReposCmd()
	case "fork-repo":
		forkRepoCmd()
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
  list-labels   List repository labels
  list-pulls    List pull requests
  create-pull   Create a pull request
  merge-pull    Merge a pull request
  view-issue    View a single issue with full details
  view-pull     View a single pull request with full details
  create-label  Create a repository label
  create-repo     Create a repository
  create-release  Create a release
  list-repos      List repositories
  fork-repo       Fork a repository
  mcp-server      Start MCP server exposing gitea-robot functionality

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
