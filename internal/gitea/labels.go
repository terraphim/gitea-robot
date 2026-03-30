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

func ResolveLabels(ctx context.Context, c client.Client, baseURL, owner, repo string, names []string) ([]int64, error) {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=50",
		baseURL, url.PathEscape(owner), url.PathEscape(repo))

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("fetching labels: %w", err)
	}

	var labels []Label
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, fmt.Errorf("parsing labels: %w", err)
	}

	nameToID := make(map[string]int64, len(labels))
	for _, l := range labels {
		nameToID[l.Name] = int64(l.ID)
	}

	var ids []int64
	for _, name := range names {
		if id, ok := nameToID[name]; ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func ListLabels(ctx context.Context, c client.Client, baseURL, owner, repo string, limit int) ([]Label, error) {
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=%d",
		baseURL, url.PathEscape(owner), url.PathEscape(repo), limit)

	data, err := c.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("listing labels: %w", err)
	}

	var labels []Label
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, fmt.Errorf("parsing labels: %w", err)
	}
	return labels, nil
}

func CreateLabel(ctx context.Context, c client.Client, baseURL, owner, repo, name, colour, description string) (*Label, error) {
	colour = normalizeColour(colour)

	payload := map[string]any{
		"name":  name,
		"color": colour,
	}
	if description != "" {
		payload["description"] = description
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling label: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels",
		baseURL, url.PathEscape(owner), url.PathEscape(repo))

	data, err := c.Post(ctx, u, body)
	if err != nil {
		return nil, fmt.Errorf("creating label: %w", err)
	}

	var label Label
	if err := json.Unmarshal(data, &label); err != nil {
		return nil, fmt.Errorf("parsing created label: %w", err)
	}
	return &label, nil
}

func normalizeColour(c string) string {
	if len(c) > 0 && c[0] != '#' {
		return "#" + c
	}
	return c
}
