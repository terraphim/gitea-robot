// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/cmdutil"
	"git.terraphim.cloud/terraphim/gitea-robot/internal/gitea"
)

func listLabelsCmd() *cobra.Command {
	var owner, repo string
	var limit int
	cmd := &cobra.Command{
		Use:   "list-labels",
		Short: "List repository labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			labels, err := gitea.ListLabels(ctx, cl, cfg.BaseURL, owner, repo, limit)
			if err != nil {
				return err
			}
			return printJSON(labels)
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of labels")
	return cmd
}

func createLabelCmd() *cobra.Command {
	var owner, repo, name, colour, description string
	cmd := &cobra.Command{
		Use:   "create-label",
		Short: "Create a repository label",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if name == "" || colour == "" {
				return fmt.Errorf("--name and --colour are required")
			}
			ctx := context.Background()
			label, err := gitea.CreateLabel(ctx, cl, cfg.BaseURL, owner, repo, name, colour, description)
			if err != nil {
				return err
			}
			fmt.Printf("Created label %d: %s (%s)\n", int64(label.ID), name, label.Color)
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&name, "name", "", "Label name")
	cmd.Flags().StringVar(&colour, "colour", "", "Label colour (hex, e.g. #FF0000)")
	cmd.Flags().StringVar(&description, "description", "", "Label description")
	return cmd
}
