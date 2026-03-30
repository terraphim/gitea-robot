// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

const (
	defaultTimeout        = 30 * time.Second
	defaultMaxResponseMB  = 10
	defaultBaseURL        = "http://localhost:3000"
)

type Config struct {
	BaseURL          string
	Token            string
	Timeout          time.Duration
	MaxResponseBytes int64
}

func LoadFromEnv() (*Config, error) {
	baseURL := os.Getenv("GITEA_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GITEA_URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Host != "localhost" && parsed.Host != "127.0.0.1" && parsed.Host != "" {
		fmt.Fprintf(os.Stderr, "Warning: GITEA_URL uses %s scheme; API token will be transmitted in plaintext\n", parsed.Scheme)
	}

	token := os.Getenv("GITEA_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITEA_TOKEN environment variable required")
	}

	return &Config{
		BaseURL:          baseURL,
		Token:            token,
		Timeout:          defaultTimeout,
		MaxResponseBytes: int64(defaultMaxResponseMB) * 1024 * 1024,
	}, nil
}
