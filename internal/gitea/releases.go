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

func CreateRelease(ctx context.Context, c client.Client, baseURL, owner, repo string, opts CreateReleaseOpts) (*Release, error) {
	title := opts.Title
	if title == "" {
		title = opts.Tag
	}

	payload := map[string]any{
		"tag_name":   opts.Tag,
		"name":       title,
		"draft":      opts.Draft,
		"prerelease": opts.Prerelease,
	}
	if opts.Body != "" {
		payload["body"] = opts.Body
	}
	if opts.Target != "" {
		payload["target_commitish"] = opts.Target
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling release: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases",
		baseURL, url.PathEscape(owner), url.PathEscape(repo))

	data, err := c.Post(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("creating release: %w", err)
	}

	var release Release
	if err := json.Unmarshal(data, &release); err != nil {
		return nil, fmt.Errorf("parsing created release: %w", err)
	}
	return &release, nil
}
