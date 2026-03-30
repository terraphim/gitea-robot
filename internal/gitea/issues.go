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

func ListIssues(ctx context.Context, c client.Client, baseURL, owner, repo string, opts ListIssueOpts) ([]Issue, error) {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=%s&limit=%d&type=issues",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), opts.State, opts.Limit)
	if opts.Labels != "" {
		u += "&labels=" + url.QueryEscape(opts.Labels)
	}

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("parsing issues: %w", err)
	}
	return issues, nil
}

func GetIssue(ctx context.Context, c client.Client, baseURL, owner, repo string, index int64) (*Issue, error) {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}
	return &issue, nil
}

func CreateIssue(ctx context.Context, c client.Client, baseURL, owner, repo string, opts CreateIssueOpts) (*Issue, error) {
	payload := map[string]any{
		"title": opts.Title,
		"body":  opts.Body,
	}

	if len(opts.Labels) > 0 {
		labelIDs, err := ResolveLabels(ctx, c, baseURL, owner, repo, opts.Labels)
		if err != nil {
			return nil, fmt.Errorf("resolving labels: %w", err)
		}
		if len(labelIDs) > 0 {
			payload["labels"] = labelIDs
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling issue: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues",
		baseURL, url.PathEscape(owner), url.PathEscape(repo))

	data, err := c.Post(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing created issue: %w", err)
	}
	return &issue, nil
}

func UpdateIssue(ctx context.Context, c client.Client, baseURL, owner, repo string, index int64, opts UpdateIssueOpts) (*Issue, error) {
	payload := map[string]any{}
	if opts.Title != "" {
		payload["title"] = opts.Title
	}
	if opts.Body != "" {
		payload["body"] = opts.Body
	}
	if opts.State != "" {
		payload["state"] = opts.State
	}

	if len(payload) > 0 {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling update: %w", err)
		}

		u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d",
			baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

		data, err := c.Patch(ctx, u, body)
		if err != nil {
			return nil, fmt.Errorf("updating issue: %w", err)
		}

		var issue Issue
		if err := json.Unmarshal(data, &issue); err != nil {
			return nil, fmt.Errorf("parsing updated issue: %w", err)
		}
		return &issue, nil
	}

	if len(opts.AddLabels) > 0 {
		labelIDs, err := ResolveLabels(ctx, c, baseURL, owner, repo, opts.AddLabels)
		if err != nil {
			return nil, fmt.Errorf("resolving labels: %w", err)
		}
		if len(labelIDs) > 0 {
			body, err := json.Marshal(map[string]any{"labels": labelIDs})
			if err != nil {
				return nil, fmt.Errorf("marshaling labels: %w", err)
			}

			u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels",
				baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

			if _, err := c.Post(ctx, u, body); err != nil {
				return nil, fmt.Errorf("adding labels: %w", err)
			}
		}
	}

	return nil, nil
}

func CloseIssue(ctx context.Context, c client.Client, baseURL, owner, repo string, index int64) (*Issue, error) {
	body := []byte(`{"state":"closed"}`)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

	data, err := c.Patch(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("closing issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing closed issue: %w", err)
	}
	return &issue, nil
}

func AddComment(ctx context.Context, c client.Client, baseURL, owner, repo string, index int64, body string) error {
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

	if _, err := c.Post(ctx, u, payload); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	return nil
}
