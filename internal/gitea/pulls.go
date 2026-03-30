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

func ListPulls(ctx context.Context, c client.Client, baseURL, owner, repo string, state string, labels string, limit int) ([]PullRequest, error) {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=%s&limit=%d",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), state, limit)
	if labels != "" {
		u += "&labels=" + url.QueryEscape(labels)
	}

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("listing pulls: %w", err)
	}

	var pulls []PullRequest
	if err := json.Unmarshal(data, &pulls); err != nil {
		return nil, fmt.Errorf("parsing pulls: %w", err)
	}
	return pulls, nil
}

func GetPull(ctx context.Context, c client.Client, baseURL, owner, repo string, index int64) (*PullRequest, error) {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting pull: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("parsing pull: %w", err)
	}
	return &pr, nil
}

func CreatePull(ctx context.Context, c client.Client, baseURL, owner, repo string, opts CreatePullOpts) (*PullRequest, error) {
	payload := map[string]any{
		"title": opts.Title,
		"head":  opts.Head,
		"base":  opts.Base,
		"body":  opts.Body,
	}
	if opts.Draft {
		payload["draft"] = true
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

	if len(opts.Assignees) > 0 {
		payload["assignees"] = opts.Assignees
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling pull: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls",
		baseURL, url.PathEscape(owner), url.PathEscape(repo))

	data, err := c.Post(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("creating pull: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("parsing created pull: %w", err)
	}
	return &pr, nil
}

func MergePull(ctx context.Context, c client.Client, baseURL, owner, repo string, index int64, opts MergePullOpts) error {
	payload := map[string]any{
		"Do":                         opts.Style,
		"delete_branch_after_merge":  opts.DeleteBranch,
	}
	if opts.Title != "" {
		payload["merge_title_field"] = opts.Title
	}
	if opts.Message != "" {
		payload["merge_message_field"] = opts.Message
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling merge: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), index)

	if _, err := c.Post(ctx, u, body); err != nil {
		return fmt.Errorf("merging pull: %w", err)
	}
	return nil
}
