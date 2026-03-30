// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/cmdutil"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/gitea"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/triage"
)

func triageCmd() *cobra.Command {
	var owner, repo, format string

	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Get prioritized task list (PageRank-ranked)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			result, err := gitea.GetTriage(ctx, cl, cfg.BaseURL, owner, repo)
			if err != nil {
				return err
			}
			if format == "markdown" {
				triage.FormatMarkdown(result, os.Stdout)
			} else {
				return printJSON(result)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&format, "format", "json", "Output format: json or markdown")
	return cmd
}

func readyCmd() *cobra.Command {
	var owner, repo string

	cmd := &cobra.Command{
		Use:   "ready",
		Short: "Get unblocked (ready) tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			data, err := gitea.GetReady(ctx, cl, cfg.BaseURL, owner, repo)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	return cmd
}

func graphCmd() *cobra.Command {
	var owner, repo string

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Get dependency graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			data, err := gitea.GetGraph(ctx, cl, cfg.BaseURL, owner, repo)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	return cmd
}

func addDepCmd() *cobra.Command {
	var owner, repo string
	var issue, blocks, relatesTo int64

	cmd := &cobra.Command{
		Use:   "add-dep",
		Short: "Add dependency between issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if issue == 0 {
				return fmt.Errorf("--issue is required")
			}
			var dependsOn int64
			var depType string
			if relatesTo > 0 {
				dependsOn = relatesTo
				depType = "relates_to"
			} else if blocks > 0 {
				dependsOn = blocks
				depType = "blocks"
			} else {
				return fmt.Errorf("--blocks or --relates-to is required")
			}
			ctx := context.Background()
			if err := gitea.AddDependency(ctx, cl, cfg.BaseURL, owner, repo, issue, dependsOn, depType); err != nil {
				return err
			}
			fmt.Println("Dependency added successfully")
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&issue, "issue", 0, "Issue index (the one being blocked)")
	cmd.Flags().Int64Var(&blocks, "blocks", 0, "Issue index that blocks this issue")
	cmd.Flags().Int64Var(&relatesTo, "relates-to", 0, "Issue index that relates to this issue")
	return cmd
}
