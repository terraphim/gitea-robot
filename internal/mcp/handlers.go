// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/client"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/gitea"
)

func (r *Registry) registerAll(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "triage",
		Description: "Get prioritized task list with PageRank scores",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":  map[string]any{"type": "string", "description": "Repository owner"},
				"repo":   map[string]any{"type": "string", "description": "Repository name"},
				"format": map[string]any{"type": "string", "description": "Output format: json or markdown", "default": "json"},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			return gitea.GetTriage(ctx, c, baseURL, owner, repo)
		},
	})

	r.register(Tool{
		Name:        "ready",
		Description: "Get unblocked (ready) tasks",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			data, err := gitea.GetReady(ctx, c, baseURL, owner, repo)
			if err != nil {
				return nil, err
			}
			return string(data), nil
		},
	})

	r.register(Tool{
		Name:        "graph",
		Description: "Get dependency graph",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			data, err := gitea.GetGraph(ctx, c, baseURL, owner, repo)
			if err != nil {
				return nil, err
			}
			return string(data), nil
		},
	})

	r.register(Tool{
		Name:        "add_dep",
		Description: "Add dependency between issues",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":      map[string]any{"type": "string", "description": "Repository owner"},
				"repo":       map[string]any{"type": "string", "description": "Repository name"},
				"issue":      map[string]any{"type": "integer", "description": "Issue ID (the one being blocked)"},
				"blocks":     map[string]any{"type": "integer", "description": "Issue ID that blocks this issue"},
				"relates_to": map[string]any{"type": "integer", "description": "Issue ID that relates to this issue"},
			},
			"required": []string{"owner", "repo", "issue"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			issue, err := requireFloat(args, "issue")
			if err != nil {
				return nil, err
			}
			if issue == 0 {
				return nil, fmt.Errorf("Missing required argument: issue")
			}

			var dependsOn int64
			var depType string
			if rt := optFloat(args, "relates_to"); rt > 0 {
				dependsOn = int64(rt)
				depType = "relates_to"
			} else if bl := optFloat(args, "blocks"); bl > 0 {
				dependsOn = int64(bl)
				depType = "blocks"
			} else {
				return nil, fmt.Errorf("Missing required argument: either blocks or relates_to must be provided")
			}

			_ = url.PathEscape("")
			if err := gitea.AddDependency(ctx, c, baseURL, owner, repo, int64(issue), dependsOn, depType); err != nil {
				return nil, err
			}
			return "Dependency added successfully", nil
		},
	})

	r.registerListLabels(c, baseURL)
	r.registerListPulls(c, baseURL)
	r.registerCreatePull(c, baseURL)
	r.registerMergePull(c, baseURL)
	r.registerViewIssue(c, baseURL)
	r.registerViewPull(c, baseURL)
	r.registerCreateLabel(c, baseURL)
	r.registerCreateRepo(c, baseURL)
	r.registerCreateRelease(c, baseURL)
	r.registerListRepos(c, baseURL)
	r.registerForkRepo(c, baseURL)
	r.registerListIssues(c, baseURL)
	r.registerCreateIssue(c, baseURL)
	r.registerComment(c, baseURL)
	r.registerCloseIssue(c, baseURL)
	r.registerEditIssue(c, baseURL)
}

func ownerRepoSchema() map[string]any {
	return map[string]any{
		"owner": map[string]any{"type": "string", "description": "Repository owner"},
		"repo":  map[string]any{"type": "string", "description": "Repository name"},
	}
}

func (r *Registry) registerListLabels(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "list_labels",
		Description: "List repository labels",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
				"limit": map[string]any{"type": "integer", "description": "Max labels", "default": 50},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			limit := optInt(args, "limit")
			if limit == 0 {
				limit = 50
			}
			return gitea.ListLabels(ctx, c, baseURL, owner, repo, limit)
		},
	})
}

func (r *Registry) registerListPulls(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "list_pulls",
		Description: "List pull requests",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":  map[string]any{"type": "string", "description": "Repository owner"},
				"repo":   map[string]any{"type": "string", "description": "Repository name"},
				"state":  map[string]any{"type": "string", "description": "PR state", "default": "open"},
				"labels": map[string]any{"type": "string", "description": "Comma-separated labels"},
				"limit":  map[string]any{"type": "integer", "description": "Max PRs", "default": 20},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			state := optString(args, "state")
			if state == "" {
				state = "open"
			}
			limit := optInt(args, "limit")
			if limit == 0 {
				limit = 20
			}
			return gitea.ListPulls(ctx, c, baseURL, owner, repo, state, optString(args, "labels"), limit)
		},
	})
}

func (r *Registry) registerCreatePull(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "create_pull",
		Description: "Create a pull request",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":     map[string]any{"type": "string", "description": "Repository owner"},
				"repo":      map[string]any{"type": "string", "description": "Repository name"},
				"title":     map[string]any{"type": "string", "description": "PR title"},
				"head":      map[string]any{"type": "string", "description": "Source branch"},
				"base":      map[string]any{"type": "string", "description": "Target branch", "default": "main"},
				"body":      map[string]any{"type": "string", "description": "PR body"},
				"labels":    map[string]any{"type": "string", "description": "Comma-separated labels"},
				"assignees": map[string]any{"type": "string", "description": "Comma-separated assignees"},
				"draft":     map[string]any{"type": "boolean", "description": "Draft PR", "default": false},
			},
			"required": []string{"owner", "repo", "title", "head"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			title, err := requireString(args, "title")
			if err != nil {
				return nil, err
			}
			head, err := requireString(args, "head")
			if err != nil {
				return nil, err
			}
			base := optString(args, "base")
			if base == "" {
				base = "main"
			}
			return gitea.CreatePull(ctx, c, baseURL, owner, repo, gitea.CreatePullOpts{
				Title:  title,
				Head:   head,
				Base:   base,
				Body:   optString(args, "body"),
				Draft:  optBool(args, "draft"),
			})
		},
	})
}

func (r *Registry) registerMergePull(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "merge_pull",
		Description: "Merge a pull request",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":         map[string]any{"type": "string", "description": "Repository owner"},
				"repo":          map[string]any{"type": "string", "description": "Repository name"},
				"index":         map[string]any{"type": "integer", "description": "PR number"},
				"style":         map[string]any{"type": "string", "description": "Merge style", "default": "merge"},
				"title":         map[string]any{"type": "string", "description": "Merge commit title"},
				"message":       map[string]any{"type": "string", "description": "Merge commit message"},
				"delete_branch": map[string]any{"type": "boolean", "description": "Delete branch", "default": false},
			},
			"required": []string{"owner", "repo", "index"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			index, err := requireFloat(args, "index")
			if err != nil {
				return nil, err
			}
			if index == 0 {
				return nil, fmt.Errorf("Missing required argument: index")
			}
			style := optString(args, "style")
			if style == "" {
				style = "merge"
			}
			err = gitea.MergePull(ctx, c, baseURL, owner, repo, int64(index), gitea.MergePullOpts{
				Style:        style,
				Title:        optString(args, "title"),
				Message:      optString(args, "message"),
				DeleteBranch: optBool(args, "delete_branch"),
			})
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("PR #%d merged (%s)", int64(index), style), nil
		},
	})
}

func (r *Registry) registerViewIssue(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "view_issue",
		Description: "View a single issue with full details",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
				"index": map[string]any{"type": "integer", "description": "Issue number"},
			},
			"required": []string{"owner", "repo", "index"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			index, err := requireFloat(args, "index")
			if err != nil {
				return nil, err
			}
			if index == 0 {
				return nil, fmt.Errorf("Missing required argument: index")
			}
			return gitea.GetIssue(ctx, c, baseURL, owner, repo, int64(index))
		},
	})
}

func (r *Registry) registerViewPull(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "view_pull",
		Description: "View a single pull request with full details",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
				"index": map[string]any{"type": "integer", "description": "PR number"},
			},
			"required": []string{"owner", "repo", "index"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			index, err := requireFloat(args, "index")
			if err != nil {
				return nil, err
			}
			if index == 0 {
				return nil, fmt.Errorf("Missing required argument: index")
			}
			return gitea.GetPull(ctx, c, baseURL, owner, repo, int64(index))
		},
	})
}

func (r *Registry) registerCreateLabel(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "create_label",
		Description: "Create a repository label",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":       map[string]any{"type": "string", "description": "Repository owner"},
				"repo":        map[string]any{"type": "string", "description": "Repository name"},
				"name":        map[string]any{"type": "string", "description": "Label name"},
				"colour":      map[string]any{"type": "string", "description": "Label colour (hex)"},
				"description": map[string]any{"type": "string", "description": "Label description"},
			},
			"required": []string{"owner", "repo", "name", "colour"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			name, err := requireString(args, "name")
			if err != nil {
				return nil, err
			}
			colour, err := requireString(args, "colour")
			if err != nil {
				return nil, err
			}
			return gitea.CreateLabel(ctx, c, baseURL, owner, repo, name, colour, optString(args, "description"))
		},
	})
}

func (r *Registry) registerCreateRepo(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "create_repo",
		Description: "Create a repository",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":           map[string]any{"type": "string", "description": "Repository name"},
				"org":            map[string]any{"type": "string", "description": "Organisation"},
				"description":    map[string]any{"type": "string", "description": "Description"},
				"private":        map[string]any{"type": "boolean", "description": "Private", "default": false},
				"auto_init":      map[string]any{"type": "boolean", "description": "Init with README", "default": false},
				"gitignore":      map[string]any{"type": "string", "description": "Gitignore template"},
				"license":        map[string]any{"type": "string", "description": "License template"},
				"default_branch": map[string]any{"type": "string", "description": "Default branch", "default": "main"},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			name, err := requireString(args, "name")
			if err != nil {
				return nil, err
			}
			db := optString(args, "default_branch")
			if db == "" {
				db = "main"
			}
			return gitea.CreateRepo(ctx, c, baseURL, gitea.CreateRepoOpts{
				Name:          name,
				Org:           optString(args, "org"),
				Description:   optString(args, "description"),
				Private:       optBool(args, "private"),
				AutoInit:      optBool(args, "auto_init"),
				Gitignore:     optString(args, "gitignore"),
				License:       optString(args, "license"),
				DefaultBranch: db,
			})
		},
	})
}

func (r *Registry) registerCreateRelease(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "create_release",
		Description: "Create a release",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":      map[string]any{"type": "string", "description": "Repository owner"},
				"repo":       map[string]any{"type": "string", "description": "Repository name"},
				"tag":        map[string]any{"type": "string", "description": "Tag name"},
				"title":      map[string]any{"type": "string", "description": "Release title"},
				"body":       map[string]any{"type": "string", "description": "Release body"},
				"target":     map[string]any{"type": "string", "description": "Target branch"},
				"draft":      map[string]any{"type": "boolean", "description": "Draft", "default": false},
				"prerelease": map[string]any{"type": "boolean", "description": "Pre-release", "default": false},
			},
			"required": []string{"owner", "repo", "tag"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			tag, err := requireString(args, "tag")
			if err != nil {
				return nil, err
			}
			return gitea.CreateRelease(ctx, c, baseURL, owner, repo, gitea.CreateReleaseOpts{
				Tag:        tag,
				Title:      optString(args, "title"),
				Body:       optString(args, "body"),
				Target:     optString(args, "target"),
				Draft:      optBool(args, "draft"),
				Prerelease: optBool(args, "prerelease"),
			})
		},
	})
}

func (r *Registry) registerListRepos(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "list_repos",
		Description: "List repositories",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"org":   map[string]any{"type": "string", "description": "Organisation"},
				"query": map[string]any{"type": "string", "description": "Search query"},
				"limit": map[string]any{"type": "integer", "description": "Max repos", "default": 20},
			},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			limit := optInt(args, "limit")
			if limit == 0 {
				limit = 20
			}
			data, err := gitea.ListRepos(ctx, c, baseURL, optString(args, "org"), optString(args, "query"), limit)
			if err != nil {
				return nil, err
			}
			return string(data), nil
		},
	})
}

func (r *Registry) registerForkRepo(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "fork_repo",
		Description: "Fork a repository",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
				"org":   map[string]any{"type": "string", "description": "Fork to organisation"},
				"name":  map[string]any{"type": "string", "description": "Fork name"},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			return gitea.ForkRepo(ctx, c, baseURL, owner, repo, gitea.ForkRepoOpts{
				Org:  optString(args, "org"),
				Name: optString(args, "name"),
			})
		},
	})
}

func (r *Registry) registerListIssues(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "list_issues",
		Description: "List repository issues",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":  map[string]any{"type": "string", "description": "Repository owner"},
				"repo":   map[string]any{"type": "string", "description": "Repository name"},
				"state":  map[string]any{"type": "string", "description": "Issue state", "default": "open"},
				"labels": map[string]any{"type": "string", "description": "Comma-separated labels"},
				"limit":  map[string]any{"type": "integer", "description": "Max issues", "default": 20},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			state := optString(args, "state")
			if state == "" {
				state = "open"
			}
			limit := optInt(args, "limit")
			if limit == 0 {
				limit = 20
			}
			return gitea.ListIssues(ctx, c, baseURL, owner, repo, gitea.ListIssueOpts{
				State:  state,
				Labels: optString(args, "labels"),
				Limit:  limit,
			})
		},
	})
}

func (r *Registry) registerCreateIssue(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "create_issue",
		Description: "Create a new issue",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":  map[string]any{"type": "string", "description": "Repository owner"},
				"repo":   map[string]any{"type": "string", "description": "Repository name"},
				"title":  map[string]any{"type": "string", "description": "Issue title"},
				"body":   map[string]any{"type": "string", "description": "Issue body"},
				"labels": map[string]any{"type": "string", "description": "Comma-separated labels"},
			},
			"required": []string{"owner", "repo", "title"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			title, err := requireString(args, "title")
			if err != nil {
				return nil, err
			}
			return gitea.CreateIssue(ctx, c, baseURL, owner, repo, gitea.CreateIssueOpts{
				Title: title,
				Body:  optString(args, "body"),
			})
		},
	})
}

func (r *Registry) registerComment(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "comment",
		Description: "Add a comment to an issue",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
				"issue": map[string]any{"type": "integer", "description": "Issue number"},
				"body":  map[string]any{"type": "string", "description": "Comment body"},
			},
			"required": []string{"owner", "repo", "issue", "body"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			issue, err := requireFloat(args, "issue")
			if err != nil {
				return nil, err
			}
			if issue == 0 {
				return nil, fmt.Errorf("Missing required argument: issue")
			}
			body, err := requireString(args, "body")
			if err != nil {
				return nil, err
			}
			if err := gitea.AddComment(ctx, c, baseURL, owner, repo, int64(issue), body); err != nil {
				return nil, err
			}
			return fmt.Sprintf("Comment added to issue #%d", int64(issue)), nil
		},
	})
}

func (r *Registry) registerCloseIssue(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "close_issue",
		Description: "Close an issue",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner": map[string]any{"type": "string", "description": "Repository owner"},
				"repo":  map[string]any{"type": "string", "description": "Repository name"},
				"issue": map[string]any{"type": "integer", "description": "Issue number"},
			},
			"required": []string{"owner", "repo", "issue"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			issue, err := requireFloat(args, "issue")
			if err != nil {
				return nil, err
			}
			if issue == 0 {
				return nil, fmt.Errorf("Missing required argument: issue")
			}
			if _, err := gitea.CloseIssue(ctx, c, baseURL, owner, repo, int64(issue)); err != nil {
				return nil, err
			}
			return fmt.Sprintf("Issue #%d closed", int64(issue)), nil
		},
	})
}

func (r *Registry) registerEditIssue(c client.Client, baseURL string) {
	r.register(Tool{
		Name:        "edit_issue",
		Description: "Edit an issue",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"owner":      map[string]any{"type": "string", "description": "Repository owner"},
				"repo":       map[string]any{"type": "string", "description": "Repository name"},
				"issue":      map[string]any{"type": "integer", "description": "Issue number"},
				"title":      map[string]any{"type": "string", "description": "New title"},
				"body":       map[string]any{"type": "string", "description": "New body"},
				"state":      map[string]any{"type": "string", "description": "New state"},
				"add_labels": map[string]any{"type": "string", "description": "Labels to add"},
			},
			"required": []string{"owner", "repo", "issue"},
		},
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			args, err := parseArgs(raw)
			if err != nil {
				return nil, err
			}
			owner, err := requireString(args, "owner")
			if err != nil {
				return nil, err
			}
			repo, err := requireString(args, "repo")
			if err != nil {
				return nil, err
			}
			issue, err := requireFloat(args, "issue")
			if err != nil {
				return nil, err
			}
			if issue == 0 {
				return nil, fmt.Errorf("Missing required argument: issue")
			}
			_, err = gitea.UpdateIssue(ctx, c, baseURL, owner, repo, int64(issue), gitea.UpdateIssueOpts{
				Title: optString(args, "title"),
				Body:  optString(args, "body"),
				State: optString(args, "state"),
			})
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("Issue #%d updated", int64(issue)), nil
		},
	})
}
