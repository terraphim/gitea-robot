// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/client"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/config"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/mcp"
)

var (
	version = "dev"
	cfg     *config.Config
	cl      client.Client
)

func main() {
	var err error
	cfg, err = config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cl = client.NewHTTPClient(cfg)

	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gitea-robot",
		Short:         "CLI for Gitea Robot API with PageRank",
		Long:          "gitea-robot - CLI for Gitea Robot API with PageRank prioritization",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		triageCmd(),
		readyCmd(),
		graphCmd(),
		addDepCmd(),
		listIssuesCmd(),
		createIssueCmd(),
		commentCmd(),
		closeIssueCmd(),
		editIssueCmd(),
		viewIssueCmd(),
		listLabelsCmd(),
		createLabelCmd(),
		listPullsCmd(),
		viewPullCmd(),
		createPullCmd(),
		mergePullCmd(),
		createRepoCmd(),
		listReposCmd(),
		forkRepoCmd(),
		createReleaseCmd(),
		mcpServerCmd(),
		versionCmd(),
	)

	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gitea-robot %s\n", version)
		},
	}
}

func mcpServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp-server",
		Short: "Start MCP server exposing gitea-robot functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := mcp.NewRegistry(cl, cfg.BaseURL)
			return mcp.RunServer(cmd.Context(), registry, os.Stdin, os.Stdout)
		},
	}
}

func encodeJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func printJSON(v any) error {
	data, err := encodeJSON(v)
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
