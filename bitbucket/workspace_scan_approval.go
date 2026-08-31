package bitbucket

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
)

func listPullRequestsByRepository(
	ctx context.Context,
	apiClient *client,
	input *URLParams,
	currentUser User,
	output io.Writer,
) ([]approvalCandidate, error) {
	rawURL, err := buildWorkspaceRepositoriesURL(
		input.Workspace,
		input.Filters.Project,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("build repository-list URL: %w", err)
	}
	repositories, err := getAllPages[Repository](ctx, apiClient, rawURL)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	if input.Filters.Project == "" {
		fmt.Fprintf(
			output,
			"Warning: scanning all %d accessible repositories in workspace %q; this may be slow.\n",
			len(repositories),
			input.Workspace,
		)
	} else {
		fmt.Fprintf(output, "Scanning %d project repositories.\n", len(repositories))
	}

	var candidates []approvalCandidate
	for _, repository := range repositories {
		if strings.TrimSpace(repository.Slug) == "" {
			return nil, fmt.Errorf("repository-list response contains an empty slug")
		}
		prURL, err := buildRepositoryPullRequestsURL(
			input.Workspace,
			repository.Slug,
			input.Filters,
			currentUser,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"build pull-request URL for repository %q: %w",
				repository.Slug,
				err,
			)
		}
		prs, err := getAllPages[PullRequest](ctx, apiClient, prURL)
		if err != nil {
			return nil, fmt.Errorf(
				"list pull requests for repository %q: %w",
				repository.Slug,
				err,
			)
		}
		for _, pr := range prs {
			candidates = append(
				candidates,
				newApprovalCandidate(pr, input.Workspace, repository.Slug),
			)
		}
	}
	return candidates, nil
}

func filterDashboardCandidates(
	candidates []approvalCandidate,
	filters DashboardFilters,
	currentUser User,
) []approvalCandidate {
	return slices.DeleteFunc(slices.Clone(candidates), func(candidate approvalCandidate) bool {
		return !matchesDashboardFilters(candidate.PullRequest, filters, currentUser)
	})
}
