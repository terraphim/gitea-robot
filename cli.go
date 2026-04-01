package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func triageCmd() {
	fs := flag.NewFlagSet("triage", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	format := fs.String("format", "json", "Output format: json or markdown")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/robot/triage?owner=%s&repo=%s", giteaURL, *owner, *repo)
	data := apiGet(url)

	if *format == "json" {
		fmt.Println(data)
	} else {
		var result map[string]any
		json.Unmarshal([]byte(data), &result)
		printTriageMarkdown(result)
	}
}

func readyCmd() {
	fs := flag.NewFlagSet("ready", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/robot/ready?owner=%s&repo=%s", giteaURL, *owner, *repo)
	data := apiGet(url)
	fmt.Println(data)
}

func graphCmd() {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/robot/graph?owner=%s&repo=%s", giteaURL, *owner, *repo)
	data := apiGet(url)
	fmt.Println(data)
}

func addDepCmd() {
	fs := flag.NewFlagSet("add-dep", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue index (the one being blocked)")
	blocks := fs.Int64("blocks", 0, "Issue index that blocks this issue")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	dependsOn := *blocks
	if dependsOn == 0 {
		fmt.Fprintln(os.Stderr, "Error: --blocks required")
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/dependencies", giteaURL, *owner, *repo, *issue)
	body := fmt.Sprintf(`{"index": %d, "owner": %q, "repo": %q}`, dependsOn, *owner, *repo)

	_, err := apiPostSafe(url, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Dependency added successfully")
}

func listIssuesCmd() {
	fs := flag.NewFlagSet("list-issues", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	state := fs.String("state", "open", "Issue state: open, closed, or all")
	labels := fs.String("labels", "", "Comma-separated label names to filter by")
	limit := fs.Int("limit", 20, "Maximum number of issues to return")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=%s&limit=%d&type=issues",
		giteaURL, *owner, *repo, *state, *limit)
	if *labels != "" {
		u += "&labels=" + *labels
	}
	data := apiGet(u)
	fmt.Println(data)
}

func createIssueCmd() {
	fs := flag.NewFlagSet("create-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	title := fs.String("title", "", "Issue title")
	body := fs.String("body", "", "Issue body")
	bodyFile := fs.String("body-file", "", "Read issue body from file")
	labels := fs.String("labels", "", "Comma-separated label names")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --title required")
		fs.Usage()
		os.Exit(1)
	}

	issueBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}

	var labelIDs []int64
	if *labels != "" {
		names := strings.Split(*labels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err = resolveLabels(*owner, *repo, names)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not resolve labels: %v\n", err)
		}
	}

	payload := map[string]any{
		"title": *title,
		"body":  issueBody,
	}
	if len(labelIDs) > 0 {
		payload["labels"] = labelIDs
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var issue map[string]any
	if err := json.Unmarshal([]byte(result), &issue); err == nil {
		if num, ok := issue["number"].(float64); ok {
			fmt.Printf("Created issue #%.0f: %s\n", num, *title)
			return
		}
	}
	fmt.Println(result)
}

func commentCmd() {
	fs := flag.NewFlagSet("comment", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue number")
	body := fs.String("body", "", "Comment body")
	bodyFile := fs.String("body-file", "", "Read comment body from file")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	commentBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}
	if commentBody == "" {
		fmt.Fprintln(os.Stderr, "Error: --body or --body-file required")
		os.Exit(1)
	}

	payload, _ := json.Marshal(map[string]string{"body": commentBody})
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", giteaURL, *owner, *repo, *issue)
	_, err = apiPostSafe(u, string(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Comment added to issue #%d\n", *issue)
}

func closeIssueCmd() {
	fs := flag.NewFlagSet("close-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue number")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	payload, _ := json.Marshal(map[string]string{"state": "closed"})
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", giteaURL, *owner, *repo, *issue)
	_, err := apiPatchSafe(u, string(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Issue #%d closed\n", *issue)
}

func editIssueCmd() {
	fs := flag.NewFlagSet("edit-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	issue := fs.Int64("issue", 0, "Issue number")
	title := fs.String("title", "", "New issue title")
	body := fs.String("body", "", "New issue body")
	bodyFile := fs.String("body-file", "", "Read issue body from file")
	state := fs.String("state", "", "New state: open or closed")
	addLabels := fs.String("add-labels", "", "Comma-separated label names to add")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *issue == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --issue required")
		fs.Usage()
		os.Exit(1)
	}

	payload := map[string]any{}
	if *title != "" {
		payload["title"] = *title
	}
	if *state != "" {
		payload["state"] = *state
	}
	issueBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}
	if issueBody != "" {
		payload["body"] = issueBody
	}

	if len(payload) > 0 {
		jsonBody, _ := json.Marshal(payload)
		u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", giteaURL, *owner, *repo, *issue)
		_, err := apiPatchSafe(u, string(jsonBody))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating issue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Issue #%d updated\n", *issue)
	}

	if *addLabels != "" {
		names := strings.Split(*addLabels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err := resolveLabels(*owner, *repo, names)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving labels: %v\n", err)
			os.Exit(1)
		}
		if len(labelIDs) > 0 {
			jsonBody, _ := json.Marshal(map[string]any{"labels": labelIDs})
			u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", giteaURL, *owner, *repo, *issue)
			_, err := apiPostSafe(u, string(jsonBody))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding labels: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Labels added to issue #%d\n", *issue)
		}
	}
}

func resolveLabels(owner, repo string, names []string) ([]int64, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=50", giteaURL, owner, repo)
	data, err := apiGetSafe(url)
	if err != nil {
		return nil, err
	}

	var labels []map[string]any
	if err := json.Unmarshal([]byte(data), &labels); err != nil {
		return nil, fmt.Errorf("error parsing labels: %v", err)
	}

	nameToID := make(map[string]int64)
	for _, l := range labels {
		name, _ := l["name"].(string)
		id, _ := l["id"].(float64)
		nameToID[name] = int64(id)
	}

	var ids []int64
	for _, name := range names {
		if id, ok := nameToID[name]; ok {
			ids = append(ids, id)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: label %q not found\n", name)
		}
	}
	return ids, nil
}

func readBody(bodyFlag, bodyFileFlag string) (string, error) {
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

func printTriageMarkdown(result map[string]any) {
	fmt.Println("## Triage Report")
	fmt.Println()

	if quickRef, ok := result["quick_ref"].(map[string]any); ok {
		fmt.Printf("**Stats:** Total: %.0f, Open: %.0f, Blocked: %.0f, Ready: %.0f\n\n",
			quickRef["total"], quickRef["open"], quickRef["blocked"], quickRef["ready"])
	}

	if recs, ok := result["recommendations"].([]any); ok {
		fmt.Println("### Top Recommendations")
		for i, r := range recs {
			if i >= 5 {
				break
			}
			rec := r.(map[string]any)
			fmt.Printf("%d. **#%.0f: %s** (PageRank: %.4f)\n",
				i+1, rec["index"], rec["title"], rec["pagerank"])
		}
	}
}

func listLabelsCmd() {
	fs := flag.NewFlagSet("list-labels", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	limit := fs.Int("limit", 50, "Maximum number of labels to return")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=%d", giteaURL, *owner, *repo, *limit)
	data := apiGet(u)
	fmt.Println(data)
}

func listPullsCmd() {
	fs := flag.NewFlagSet("list-pulls", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	state := fs.String("state", "open", "PR state: open, closed, or all")
	labels := fs.String("labels", "", "Comma-separated label names to filter by")
	limit := fs.Int("limit", 20, "Maximum number of PRs to return")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=%s&limit=%d",
		giteaURL, *owner, *repo, *state, *limit)
	if *labels != "" {
		u += "&labels=" + *labels
	}
	data := apiGet(u)
	fmt.Println(data)
}

func createPullCmd() {
	fs := flag.NewFlagSet("create-pull", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	title := fs.String("title", "", "PR title")
	head := fs.String("head", "", "Source branch")
	base := fs.String("base", "main", "Target branch")
	body := fs.String("body", "", "PR body")
	bodyFile := fs.String("body-file", "", "Read PR body from file")
	labels := fs.String("labels", "", "Comma-separated label names")
	assignees := fs.String("assignees", "", "Comma-separated assignee usernames")
	draft := fs.Bool("draft", false, "Create as draft PR")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *title == "" || *head == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, --title, and --head required")
		fs.Usage()
		os.Exit(1)
	}

	prBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}

	payload := map[string]any{
		"title": *title,
		"head":  *head,
		"base":  *base,
		"body":  prBody,
	}

	if *draft {
		payload["draft"] = true
	}

	if *labels != "" {
		names := strings.Split(*labels, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		labelIDs, err := resolveLabels(*owner, *repo, names)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not resolve labels: %v\n", err)
		}
		if len(labelIDs) > 0 {
			payload["labels"] = labelIDs
		}
	}

	if *assignees != "" {
		names := strings.Split(*assignees, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		payload["assignees"] = names
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var pr map[string]any
	if err := json.Unmarshal([]byte(result), &pr); err == nil {
		if num, ok := pr["number"].(float64); ok {
			fmt.Printf("Created PR #%.0f: %s\n", num, *title)
			if url, ok := pr["html_url"].(string); ok {
				fmt.Println(url)
			}
			return
		}
	}
	fmt.Println(result)
}

func mergePullCmd() {
	fs := flag.NewFlagSet("merge-pull", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	index := fs.Int64("index", 0, "PR number")
	style := fs.String("style", "merge", "Merge style: merge, rebase, or squash")
	mergeTitle := fs.String("title", "", "Merge commit title")
	mergeMsg := fs.String("message", "", "Merge commit message")
	deleteBranch := fs.Bool("delete-branch", false, "Delete source branch after merge")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *index == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --index required")
		fs.Usage()
		os.Exit(1)
	}

	payload := map[string]any{
		"Do":                        *style,
		"delete_branch_after_merge": *deleteBranch,
	}
	if *mergeTitle != "" {
		payload["merge_title_field"] = *mergeTitle
	}
	if *mergeMsg != "" {
		payload["merge_message_field"] = *mergeMsg
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge", giteaURL, *owner, *repo, *index)
	_, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("PR #%d merged (%s)\n", *index, *style)
}

func viewIssueCmd() {
	fs := flag.NewFlagSet("view-issue", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	index := fs.Int64("index", 0, "Issue number")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *index == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --index required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", giteaURL, *owner, *repo, *index)
	data := apiGet(u)
	fmt.Println(data)
}

func viewPullCmd() {
	fs := flag.NewFlagSet("view-pull", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	index := fs.Int64("index", 0, "PR number")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *index == 0 {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --index required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d", giteaURL, *owner, *repo, *index)
	data := apiGet(u)
	fmt.Println(data)
}

func createLabelCmd() {
	fs := flag.NewFlagSet("create-label", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	name := fs.String("name", "", "Label name")
	colour := fs.String("colour", "", "Label colour (hex, e.g. #FF0000)")
	description := fs.String("description", "", "Label description")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *name == "" || *colour == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, --name, and --colour required")
		fs.Usage()
		os.Exit(1)
	}

	// Normalise colour: ensure # prefix
	c := *colour
	if len(c) > 0 && c[0] != '#' {
		c = "#" + c
	}

	payload := map[string]any{
		"name":  *name,
		"color": c,
	}
	if *description != "" {
		payload["description"] = *description
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var label map[string]any
	if err := json.Unmarshal([]byte(result), &label); err == nil {
		if id, ok := label["id"].(float64); ok {
			fmt.Printf("Created label %.0f: %s (%s)\n", id, *name, c)
			return
		}
	}
	fmt.Println(result)
}

func createRepoCmd() {
	fs := flag.NewFlagSet("create-repo", flag.ExitOnError)
	name := fs.String("name", "", "Repository name")
	org := fs.String("org", "", "Organisation (omit for personal repo)")
	description := fs.String("description", "", "Repository description")
	private := fs.Bool("private", false, "Create as private repository")
	autoInit := fs.Bool("auto-init", false, "Initialise with README")
	gitignore := fs.String("gitignore", "", "Gitignore template (e.g. Go)")
	license := fs.String("license", "", "License template (e.g. MIT)")
	defaultBranch := fs.String("default-branch", "main", "Default branch name")
	fs.Parse(os.Args[1:])

	if *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --name required")
		fs.Usage()
		os.Exit(1)
	}

	payload := map[string]any{
		"name":           *name,
		"private":        *private,
		"auto_init":      *autoInit,
		"default_branch": *defaultBranch,
	}
	if *description != "" {
		payload["description"] = *description
	}
	if *gitignore != "" {
		payload["gitignores"] = *gitignore
	}
	if *license != "" {
		payload["license"] = *license
	}

	var u string
	if *org != "" {
		u = fmt.Sprintf("%s/api/v1/orgs/%s/repos", giteaURL, *org)
	} else {
		u = fmt.Sprintf("%s/api/v1/user/repos", giteaURL)
	}

	jsonBody, _ := json.Marshal(payload)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var repo map[string]any
	if err := json.Unmarshal([]byte(result), &repo); err == nil {
		if fullName, ok := repo["full_name"].(string); ok {
			fmt.Printf("Created repository: %s\n", fullName)
			if htmlURL, ok := repo["html_url"].(string); ok {
				fmt.Println(htmlURL)
			}
			return
		}
	}
	fmt.Println(result)
}

func createReleaseCmd() {
	fs := flag.NewFlagSet("create-release", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	tag := fs.String("tag", "", "Tag name (e.g. v1.0.0)")
	title := fs.String("title", "", "Release title")
	body := fs.String("body", "", "Release body")
	bodyFile := fs.String("body-file", "", "Read release body from file")
	target := fs.String("target", "", "Target branch (default: repo default branch)")
	draft := fs.Bool("draft", false, "Create as draft release")
	prerelease := fs.Bool("prerelease", false, "Mark as pre-release")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *tag == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --tag required")
		fs.Usage()
		os.Exit(1)
	}

	releaseBody, err := readBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}

	payload := map[string]any{
		"tag_name":   *tag,
		"draft":      *draft,
		"prerelease": *prerelease,
	}
	if *title != "" {
		payload["name"] = *title
	} else {
		payload["name"] = *tag
	}
	if releaseBody != "" {
		payload["body"] = releaseBody
	}
	if *target != "" {
		payload["target_commitish"] = *target
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var release map[string]any
	if err := json.Unmarshal([]byte(result), &release); err == nil {
		if tagName, ok := release["tag_name"].(string); ok {
			fmt.Printf("Created release: %s\n", tagName)
			if htmlURL, ok := release["html_url"].(string); ok {
				fmt.Println(htmlURL)
			}
			return
		}
	}
	fmt.Println(result)
}

func listReposCmd() {
	fs := flag.NewFlagSet("list-repos", flag.ExitOnError)
	org := fs.String("org", "", "Organisation name")
	limit := fs.Int("limit", 20, "Maximum number of repos to return")
	query := fs.String("query", "", "Search query")
	fs.Parse(os.Args[1:])

	var u string
	if *org != "" {
		u = fmt.Sprintf("%s/api/v1/orgs/%s/repos?limit=%d", giteaURL, *org, *limit)
	} else if *query != "" {
		u = fmt.Sprintf("%s/api/v1/repos/search?q=%s&limit=%d", giteaURL, *query, *limit)
	} else {
		u = fmt.Sprintf("%s/api/v1/repos/search?limit=%d", giteaURL, *limit)
	}
	data := apiGet(u)
	fmt.Println(data)
}

func forkRepoCmd() {
	fs := flag.NewFlagSet("fork-repo", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	org := fs.String("org", "", "Fork to organisation (omit for personal fork)")
	name := fs.String("name", "", "Fork name (defaults to original)")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	payload := map[string]any{}
	if *org != "" {
		payload["organization"] = *org
	}
	if *name != "" {
		payload["name"] = *name
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/forks", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var forked map[string]any
	if err := json.Unmarshal([]byte(result), &forked); err == nil {
		if fullName, ok := forked["full_name"].(string); ok {
			fmt.Printf("Forked to: %s\n", fullName)
			if htmlURL, ok := forked["html_url"].(string); ok {
				fmt.Println(htmlURL)
			}
			return
		}
	}
	fmt.Println(result)
}

// Wiki commands

func wikiCreateCmd() {
	fs := flag.NewFlagSet("wiki-create", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	title := fs.String("title", "", "Wiki page title")
	content := fs.String("content", "", "Wiki page content (markdown)")
	file := fs.String("file", "", "Read content from file (alternative to --content)")
	message := fs.String("message", "", "Commit message (default: \"Create wiki page: {title}\")")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --title required")
		fs.Usage()
		os.Exit(1)
	}

	var pageContent string
	var err error
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		pageContent = string(data)
	} else if *content != "" {
		pageContent = *content
	} else {
		fmt.Fprintln(os.Stderr, "Error: --content or --file required")
		fs.Usage()
		os.Exit(1)
	}

	commitMsg := *message
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Create wiki page: %s", *title)
	}

	contentBase64 := base64.StdEncoding.EncodeToString([]byte(pageContent))
	payload := map[string]any{
		"title":          *title,
		"content_base64": contentBase64,
		"message":        commitMsg,
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/new", giteaURL, *owner, *repo)
	result, err := apiPostSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}

func wikiListCmd() {
	fs := flag.NewFlagSet("wiki-list", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner and --repo required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/pages", giteaURL, *owner, *repo)
	data := apiGet(u)
	fmt.Println(data)
}

func wikiGetCmd() {
	fs := flag.NewFlagSet("wiki-get", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	name := fs.String("name", "", "Wiki page name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --name required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/page/%s", giteaURL, *owner, *repo, *name)
	data := apiGet(u)

	// Decode base64 content for user-readable output
	var page map[string]any
	if err := json.Unmarshal([]byte(data), &page); err == nil {
		if contentBase64, ok := page["content_base64"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(contentBase64)
			if err == nil {
				page["content"] = string(decoded)
				delete(page, "content_base64")
			}
		}
		decodedJSON, _ := json.Marshal(page)
		fmt.Println(string(decodedJSON))
	} else {
		fmt.Println(data)
	}
}

func wikiUpdateCmd() {
	fs := flag.NewFlagSet("wiki-update", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	name := fs.String("name", "", "Wiki page name")
	content := fs.String("content", "", "Wiki page content (markdown)")
	file := fs.String("file", "", "Read content from file (alternative to --content)")
	message := fs.String("message", "", "Commit message (default: \"Update wiki page: {name}\")")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --name required")
		fs.Usage()
		os.Exit(1)
	}

	var pageContent string
	var err error
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		pageContent = string(data)
	} else if *content != "" {
		pageContent = *content
	} else {
		fmt.Fprintln(os.Stderr, "Error: --content or --file required")
		fs.Usage()
		os.Exit(1)
	}

	commitMsg := *message
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Update wiki page: %s", *name)
	}

	contentBase64 := base64.StdEncoding.EncodeToString([]byte(pageContent))
	payload := map[string]any{
		"content_base64": contentBase64,
		"message":        commitMsg,
	}

	jsonBody, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/page/%s", giteaURL, *owner, *repo, *name)
	result, err := apiPatchSafe(u, string(jsonBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}

func wikiDeleteCmd() {
	fs := flag.NewFlagSet("wiki-delete", flag.ExitOnError)
	owner := fs.String("owner", "", "Repository owner")
	repo := fs.String("repo", "", "Repository name")
	name := fs.String("name", "", "Wiki page name")
	fs.Parse(os.Args[1:])

	if *owner == "" || *repo == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo, and --name required")
		fs.Usage()
		os.Exit(1)
	}

	u := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/page/%s", giteaURL, *owner, *repo, *name)
	_, err := apiDeleteSafe(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wiki page '%s' deleted\n", *name)
}
