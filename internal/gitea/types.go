// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitea

type TriageResult struct {
	QuickRef       QuickRef        `json:"quick_ref"`
	Recommendations []Recommendation `json:"recommendations"`
}

type QuickRef struct {
	Total  float64 `json:"total"`
	Open   float64 `json:"open"`
	Blocked float64 `json:"blocked"`
	Ready  float64 `json:"ready"`
}

type Recommendation struct {
	Index    float64 `json:"index"`
	Title    string  `json:"title"`
	PageRank float64 `json:"pagerank"`
}

type Issue struct {
	Number  float64 `json:"number"`
	Title   string  `json:"title"`
	Body    string  `json:"body"`
	State   string  `json:"state"`
	HTMLURL string  `json:"html_url"`
}

type PullRequest struct {
	Number  float64 `json:"number"`
	Title   string  `json:"title"`
	Body    string  `json:"body"`
	State   string  `json:"state"`
	HTMLURL string  `json:"html_url"`
	Head    PRBranch `json:"head"`
	Base    PRBranch `json:"base"`
}

type PRBranch struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
	Sha   string `json:"sha"`
}

type Label struct {
	ID          float64 `json:"id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Description string  `json:"description"`
}

type Repo struct {
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type ListIssueOpts struct {
	State  string
	Labels string
	Limit  int
}

type CreateIssueOpts struct {
	Title  string
	Body   string
	Labels []string
}

type UpdateIssueOpts struct {
	Title     string
	Body      string
	State     string
	AddLabels []string
}

type CreatePullOpts struct {
	Title     string
	Head      string
	Base      string
	Body      string
	Labels    []string
	Assignees []string
	Draft     bool
}

type MergePullOpts struct {
	Style         string
	Title         string
	Message       string
	DeleteBranch  bool
}

type CreateRepoOpts struct {
	Name          string
	Org           string
	Description   string
	Private       bool
	AutoInit      bool
	Gitignore     string
	License       string
	DefaultBranch string
}

type CreateReleaseOpts struct {
	Tag        string
	Title      string
	Body       string
	Target     string
	Draft      bool
	Prerelease bool
}

type ForkRepoOpts struct {
	Org  string
	Name string
}
