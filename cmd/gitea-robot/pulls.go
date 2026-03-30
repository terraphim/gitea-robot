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

func listPullsCmd() *cobra.Command {
	var owner, repo, state, labels string
	var limit int
	cmd := &cobra.Command{
		Use:   "list-pulls",
		Short: "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			pulls, err := gitea.ListPulls(ctx, cl, cfg.BaseURL, owner, repo, state, labels, limit)
			if err != nil {
				return err
			}
			return printJSON(pulls)
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&state, "state", "open", "PR state: open, closed, all")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated label names")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of PRs")
	return cmd
}

func viewPullCmd() *cobra.Command {
	var owner, repo string
	var index int64
	cmd := &cobra.Command{
		Use:   "view-pull",
		Short: "View a single pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if index == 0 {
				return fmt.Errorf("--index is required")
			}
			ctx := context.Background()
			pr, err := gitea.GetPull(ctx, cl, cfg.BaseURL, owner, repo, index)
			if err != nil {
				return err
			}
			return printJSON(pr)
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&index, "index", 0, "PR number")
	return cmd
}

func createPullCmd() *cobra.Command {
	var owner, repo, title, head, base, body, bodyFile, labels, assignees string
	var draft bool
	cmd := &cobra.Command{
		Use:   "create-pull",
		Short: "Create a pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if title == "" || head == "" {
				return fmt.Errorf("--title and --head are required")
			}
			prBody, err := cmdutil.ReadBody(body, bodyFile)
			if err != nil {
				return fmt.Errorf("reading body: %w", err)
			}
			ctx := context.Background()
			pr, err := gitea.CreatePull(ctx, cl, cfg.BaseURL, owner, repo, gitea.CreatePullOpts{
				Title: title, Head: head, Base: base, Body: prBody,
				Labels: cmdutil.SplitLabelNames(labels), Assignees: cmdutil.SplitLabelNames(assignees), Draft: draft,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created PR #%d: %s\n", int64(pr.Number), title)
			if pr.HTMLURL != "" {
				fmt.Println(pr.HTMLURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&title, "title", "", "PR title")
	cmd.Flags().StringVar(&head, "head", "", "Source branch")
	cmd.Flags().StringVar(&base, "base", "main", "Target branch")
	cmd.Flags().StringVar(&body, "body", "", "PR body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read body from file")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated label names")
	cmd.Flags().StringVar(&assignees, "assignees", "", "Comma-separated assignee usernames")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create as draft PR")
	return cmd
}

func mergePullCmd() *cobra.Command {
	var owner, repo string
	var index int64
	var style, mergeTitle, mergeMsg string
	var deleteBranch bool
	cmd := &cobra.Command{
		Use:   "merge-pull",
		Short: "Merge a pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if index == 0 {
				return fmt.Errorf("--index is required")
			}
			ctx := context.Background()
			if err := gitea.MergePull(ctx, cl, cfg.BaseURL, owner, repo, index, gitea.MergePullOpts{
				Style: style, Title: mergeTitle, Message: mergeMsg, DeleteBranch: deleteBranch,
			}); err != nil {
				return err
			}
			fmt.Printf("PR #%d merged (%s)\n", index, style)
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&index, "index", 0, "PR number")
	cmd.Flags().StringVar(&style, "style", "merge", "Merge style: merge, rebase, squash")
	cmd.Flags().StringVar(&mergeTitle, "title", "", "Merge commit title")
	cmd.Flags().StringVar(&mergeMsg, "message", "", "Merge commit message")
	cmd.Flags().BoolVar(&deleteBranch, "delete-branch", false, "Delete source branch after merge")
	return cmd
}
