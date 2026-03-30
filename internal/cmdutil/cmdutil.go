// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func RequireOwnerRepo(owner, repo string) error {
	if owner == "" {
		return fmt.Errorf("--owner is required")
	}
	if repo == "" {
		return fmt.Errorf("--repo is required")
	}
	return nil
}

func ReadBody(bodyFlag, bodyFileFlag string) (string, error) {
	if bodyFileFlag == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}
	if bodyFileFlag != "" {
		data, err := os.ReadFile(bodyFileFlag)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return bodyFlag, nil
}

func SplitLabelNames(labels string) []string {
	if labels == "" {
		return nil
	}
	parts := strings.Split(labels, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
