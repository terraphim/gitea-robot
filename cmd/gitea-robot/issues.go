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

func listIssuesCmd() *cobra.Command {
	var owner, repo, state, labels string
	var limit int
	cmd := &cobra.Command{
		Use:   "list-issues",
		Short: "List repository issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			issues, err := gitea.ListIssues(ctx, cl, cfg.BaseURL, owner, repo, gitea.ListIssueOpts{
				State: state, Labels: labels, Limit: limit,
			})
			if err != nil {
				return err
			}
			return printJSON(issues)
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&state, "state", "open", "Issue state: open, closed, all")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated label names")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of issues")
	return cmd
}

func createIssueCmd() *cobra.Command {
	var owner, repo, title, body, bodyFile, labels string
	cmd := &cobra.Command{
		Use:   "create-issue",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			issueBody, err := cmdutil.ReadBody(body, bodyFile)
			if err != nil {
				return fmt.Errorf("reading body: %w", err)
			}
			ctx := context.Background()
			issue, err := gitea.CreateIssue(ctx, cl, cfg.BaseURL, owner, repo, gitea.CreateIssueOpts{
				Title: title, Body: issueBody, Labels: cmdutil.SplitLabelNames(labels),
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created issue #%d: %s\n", int64(issue.Number), title)
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&title, "title", "", "Issue title")
	cmd.Flags().StringVar(&body, "body", "", "Issue body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read body from file")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated label names")
	return cmd
}

func commentCmd() *cobra.Command {
	var owner, repo, body, bodyFile string
	var issue int64
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Add a comment to an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if issue == 0 {
				return fmt.Errorf("--issue is required")
			}
			commentBody, err := cmdutil.ReadBody(body, bodyFile)
			if err != nil {
				return fmt.Errorf("reading body: %w", err)
			}
			if commentBody == "" {
				return fmt.Errorf("--body or --body-file is required")
			}
			ctx := context.Background()
			if err := gitea.AddComment(ctx, cl, cfg.BaseURL, owner, repo, issue, commentBody); err != nil {
				return err
			}
			fmt.Printf("Comment added to issue #%d\n", issue)
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&issue, "issue", 0, "Issue number")
	cmd.Flags().StringVar(&body, "body", "", "Comment body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read body from file")
	return cmd
}

func closeIssueCmd() *cobra.Command {
	var owner, repo string
	var issue int64
	cmd := &cobra.Command{
		Use:   "close-issue",
		Short: "Close an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if issue == 0 {
				return fmt.Errorf("--issue is required")
			}
			ctx := context.Background()
			if _, err := gitea.CloseIssue(ctx, cl, cfg.BaseURL, owner, repo, issue); err != nil {
				return err
			}
			fmt.Printf("Issue #%d closed\n", issue)
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&issue, "issue", 0, "Issue number")
	return cmd
}

func editIssueCmd() *cobra.Command {
	var owner, repo, title, body, bodyFile, state, addLabels string
	var issue int64
	cmd := &cobra.Command{
		Use:   "edit-issue",
		Short: "Edit an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if issue == 0 {
				return fmt.Errorf("--issue is required")
			}
			issueBody, err := cmdutil.ReadBody(body, bodyFile)
			if err != nil {
				return fmt.Errorf("reading body: %w", err)
			}
			ctx := context.Background()
			_, err = gitea.UpdateIssue(ctx, cl, cfg.BaseURL, owner, repo, issue, gitea.UpdateIssueOpts{
				Title: title, Body: issueBody, State: state, AddLabels: cmdutil.SplitLabelNames(addLabels),
			})
			if err != nil {
				return err
			}
			fmt.Printf("Issue #%d updated\n", issue)
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&issue, "issue", 0, "Issue number")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&body, "body", "", "New body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read body from file")
	cmd.Flags().StringVar(&state, "state", "", "New state: open or closed")
	cmd.Flags().StringVar(&addLabels, "add-labels", "", "Comma-separated labels to add")
	return cmd
}

func viewIssueCmd() *cobra.Command {
	var owner, repo string
	var index int64
	cmd := &cobra.Command{
		Use:   "view-issue",
		Short: "View a single issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if index == 0 {
				return fmt.Errorf("--index is required")
			}
			ctx := context.Background()
			issue, err := gitea.GetIssue(ctx, cl, cfg.BaseURL, owner, repo, index)
			if err != nil {
				return err
			}
			return printJSON(issue)
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().Int64Var(&index, "index", 0, "Issue number")
	return cmd
}
