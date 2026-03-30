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

func GetTriage(ctx context.Context, c client.Client, baseURL, owner, repo string) (*TriageResult, error) {
	u := fmt.Sprintf("%s/api/v1/robot/triage?owner=%s&repo=%s",
		baseURL, url.QueryEscape(owner), url.QueryEscape(repo))

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting triage: %w", err)
	}

	var result TriageResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing triage response: %w", err)
	}
	return &result, nil
}

func GetReady(ctx context.Context, c client.Client, baseURL, owner, repo string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/robot/ready?owner=%s&repo=%s",
		baseURL, url.QueryEscape(owner), url.QueryEscape(repo))

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting ready issues: %w", err)
	}
	return json.RawMessage(data), nil
}

func GetGraph(ctx context.Context, c client.Client, baseURL, owner, repo string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/robot/graph?owner=%s&repo=%s",
		baseURL, url.QueryEscape(owner), url.QueryEscape(repo))

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting dependency graph: %w", err)
	}
	return json.RawMessage(data), nil
}

func AddDependency(ctx context.Context, c client.Client, baseURL, owner, repo string, issue, dependsOn int64, depType string) error {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/dependencies",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), issue)

	payload := map[string]any{
		"index": dependsOn,
		"owner": owner,
		"repo":  repo,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling dependency request: %w", err)
	}

	if _, err := c.Post(ctx, u, body); err != nil {
		return fmt.Errorf("adding dependency: %w", err)
	}
	return nil
}
