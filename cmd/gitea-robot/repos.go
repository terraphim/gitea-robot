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

func createRepoCmd() *cobra.Command {
	var name, org, description, gitignore, license, defaultBranch string
	var private, autoInit bool
	cmd := &cobra.Command{
		Use:   "create-repo",
		Short: "Create a repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			ctx := context.Background()
			repo, err := gitea.CreateRepo(ctx, cl, cfg.BaseURL, gitea.CreateRepoOpts{
				Name: name, Org: org, Description: description, Private: private,
				AutoInit: autoInit, Gitignore: gitignore, License: license, DefaultBranch: defaultBranch,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created repository: %s\n", repo.FullName)
			if repo.HTMLURL != "" {
				fmt.Println(repo.HTMLURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Repository name")
	cmd.Flags().StringVar(&org, "org", "", "Organisation (omit for personal)")
	cmd.Flags().StringVar(&description, "description", "", "Repository description")
	cmd.Flags().BoolVar(&private, "private", false, "Create as private")
	cmd.Flags().BoolVar(&autoInit, "auto-init", false, "Initialise with README")
	cmd.Flags().StringVar(&gitignore, "gitignore", "", "Gitignore template")
	cmd.Flags().StringVar(&license, "license", "", "License template")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "main", "Default branch name")
	return cmd
}

func listReposCmd() *cobra.Command {
	var org, query string
	var limit int
	cmd := &cobra.Command{
		Use:   "list-repos",
		Short: "List repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			data, err := gitea.ListRepos(ctx, cl, cfg.BaseURL, org, query, limit)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&org, "org", "", "Organisation name")
	cmd.Flags().StringVar(&query, "query", "", "Search query")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of repos")
	return cmd
}

func forkRepoCmd() *cobra.Command {
	var owner, repo, org, name string
	cmd := &cobra.Command{
		Use:   "fork-repo",
		Short: "Fork a repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			ctx := context.Background()
			forked, err := gitea.ForkRepo(ctx, cl, cfg.BaseURL, owner, repo, gitea.ForkRepoOpts{
				Org: org, Name: name,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Forked to: %s\n", forked.FullName)
			if forked.HTMLURL != "" {
				fmt.Println(forked.HTMLURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&org, "org", "", "Fork to organisation")
	cmd.Flags().StringVar(&name, "name", "", "Fork name")
	return cmd
}

func createReleaseCmd() *cobra.Command {
	var owner, repo, tag, title, body, bodyFile, target string
	var draft, prerelease bool
	cmd := &cobra.Command{
		Use:   "create-release",
		Short: "Create a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireOwnerRepo(owner, repo); err != nil {
				return err
			}
			if tag == "" {
				return fmt.Errorf("--tag is required")
			}
			releaseBody, err := cmdutil.ReadBody(body, bodyFile)
			if err != nil {
				return fmt.Errorf("reading body: %w", err)
			}
			ctx := context.Background()
			release, err := gitea.CreateRelease(ctx, cl, cfg.BaseURL, owner, repo, gitea.CreateReleaseOpts{
				Tag: tag, Title: title, Body: releaseBody, Target: target, Draft: draft, Prerelease: prerelease,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created release: %s\n", release.TagName)
			if release.HTMLURL != "" {
				fmt.Println(release.HTMLURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&tag, "tag", "", "Tag name (e.g. v1.0.0)")
	cmd.Flags().StringVar(&title, "title", "", "Release title")
	cmd.Flags().StringVar(&body, "body", "", "Release body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read body from file")
	cmd.Flags().StringVar(&target, "target", "", "Target branch")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create as draft")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Mark as pre-release")
	return cmd
}
