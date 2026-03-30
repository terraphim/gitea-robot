// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package triage

import (
	"fmt"
	"io"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/gitea"
)

const maxRecommendations = 10

func FormatMarkdown(result *gitea.TriageResult, w io.Writer) {
	fmt.Fprintln(w, "## Triage Report")
	fmt.Fprintln(w)

	qr := result.QuickRef
	fmt.Fprintf(w, "**Stats:** Total: %.0f, Open: %.0f, Blocked: %.0f, Ready: %.0f\n\n",
		qr.Total, qr.Open, qr.Blocked, qr.Ready)

	if len(result.Recommendations) > 0 {
		fmt.Fprintln(w, "### Top Recommendations")
		count := len(result.Recommendations)
		if count > maxRecommendations {
			count = maxRecommendations
		}
		for i := 0; i < count; i++ {
			rec := result.Recommendations[i]
			fmt.Fprintf(w, "%d. **#%.0f: %s** (PageRank: %.4f)\n",
				i+1, rec.Index, rec.Title, rec.PageRank)
		}
	}
}
