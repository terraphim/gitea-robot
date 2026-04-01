// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// gitea-robot CLI - thin wrapper for Gitea Robot API

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	giteaURL   = os.Getenv("GITEA_URL")
	giteaToken = os.Getenv("GITEA_TOKEN")
)

func main() {
	if giteaURL == "" {
		giteaURL = "http://localhost:3000"
	}

	// Set global HTTP client timeout to prevent MCP server hangs
	http.DefaultClient.Timeout = 30 * time.Second

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
	case "wiki-create":
		wikiCreateCmd()
	case "wiki-list":
		wikiListCmd()
	case "wiki-get":
		wikiGetCmd()
	case "wiki-update":
		wikiUpdateCmd()
	case "wiki-delete":
		wikiDeleteCmd()
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
  wiki-create     Create a wiki page
  wiki-list       List wiki pages
  wiki-get        Get a wiki page with decoded content
  wiki-update     Update a wiki page
  wiki-delete     Delete a wiki page

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

  # Create a wiki page
  gitea-robot wiki-create --owner terraphim --repo gitea --title "Home" --content "# Welcome"

  # List wiki pages
  gitea-robot wiki-list --owner terraphim --repo gitea

  # Get a wiki page
  gitea-robot wiki-get --owner terraphim --repo gitea --name "Home"

  # Update a wiki page from file
  gitea-robot wiki-update --owner terraphim --repo gitea --name "Home" --file docs/home.md

  # Delete a wiki page
  gitea-robot wiki-delete --owner terraphim --repo gitea --name "Home"

  # Start MCP server
  gitea-robot mcp-server`)
}
