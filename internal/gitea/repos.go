// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/client"
)

func ListRepos(ctx context.Context, c client.Client, baseURL, org, query string, limit int) (json.RawMessage, error) {
	var u string
	if org != "" {
		u = fmt.Sprintf("%s/api/v1/orgs/%s/repos?limit=%d",
			baseURL, url.PathEscape(org), limit)
	} else if query != "" {
		u = fmt.Sprintf("%s/api/v1/repos/search?q=%s&limit=%d",
			baseURL, url.QueryEscape(query), limit)
	} else {
		u = fmt.Sprintf("%s/api/v1/repos/search?limit=%d", baseURL, limit)
	}

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("listing repos: %w", err)
	}
	return json.RawMessage(data), nil
}

func CreateRepo(ctx context.Context, c client.Client, baseURL string, opts CreateRepoOpts) (*Repo, error) {
	payload := map[string]any{
		"name":           opts.Name,
		"private":        opts.Private,
		"auto_init":      opts.AutoInit,
		"default_branch": opts.DefaultBranch,
	}
	if opts.Description != "" {
		payload["description"] = opts.Description
	}
	if opts.Gitignore != "" {
		payload["gitignores"] = opts.Gitignore
	}
	if opts.License != "" {
		payload["license"] = opts.License
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling repo: %w", err)
	}

	var u string
	if opts.Org != "" {
		u = fmt.Sprintf("%s/api/v1/orgs/%s/repos", baseURL, url.PathEscape(opts.Org))
	} else {
		u = fmt.Sprintf("%s/api/v1/user/repos", baseURL)
	}

	data, err := c.Post(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("creating repo: %w", err)
	}

	var repo Repo
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("parsing created repo: %w", err)
	}
	return &repo, nil
}

func ForkRepo(ctx context.Context, c client.Client, baseURL, owner, repo string, opts ForkRepoOpts) (*Repo, error) {
	payload := map[string]any{}
	if opts.Org != "" {
		payload["organization"] = opts.Org
	}
	if opts.Name != "" {
		payload["name"] = opts.Name
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling fork: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/forks",
		baseURL, url.PathEscape(owner), url.PathEscape(repo))

	data, err := c.Post(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("forking repo: %w", err)
	}

	var forked Repo
	if err := json.Unmarshal(data, &forked); err != nil {
		return nil, fmt.Errorf("parsing forked repo: %w", err)
	}
	return &forked, nil
}
