// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/config"
)

type Client interface {
	Get(ctx context.Context, url string) ([]byte, error)
	Post(ctx context.Context, url string, body []byte) ([]byte, error)
	Patch(ctx context.Context, url string, body []byte) ([]byte, error)
	Delete(ctx context.Context, url string) ([]byte, error)
}

type httpClient struct {
	httpClient *http.Client
	token      string
	maxBytes   int64
}

func NewHTTPClient(cfg *config.Config) Client {
	return &httpClient{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		token:    cfg.Token,
		maxBytes: cfg.MaxResponseBytes,
	}
}

func (c *httpClient) Get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request: %w", err)
	}
	c.setAuth(req)
	return c.do(req)
}

func (c *httpClient) Post(ctx context.Context, rawURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("creating POST request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *httpClient) Patch(ctx context.Context, rawURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, rawURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("creating PATCH request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *httpClient) Delete(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating DELETE request: %w", err)
	}
	c.setAuth(req)
	return c.do(req)
}

func (c *httpClient) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "token "+c.token)
}

func (c *httpClient) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	reader := io.LimitReader(resp.Body, c.maxBytes)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %s: %s", resp.Status, string(body))
	}

	return body, nil
}
