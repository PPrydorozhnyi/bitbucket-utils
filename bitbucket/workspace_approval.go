package bitbucket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func listPullRequestsByAuthor(
	ctx context.Context,
	apiClient *client,
	input *URLParams,
	currentUser User,
	output io.Writer,
) ([]approvalCandidate, error) {
	prs, err := listAuthor(ctx, apiClient, input, currentUser, input.Filters.Author, output)
	if isAPIStatus(err, http.StatusNotFound) {
		_, suffix, found := strings.Cut(input.Filters.Author, ":")
		if found && suffix != "" {
			fmt.Fprintf(
				output,
				"Author %q was not found; retrying with UUID suffix %q.\n",
				input.Filters.Author,
				suffix,
			)
			prs, err = listAuthor(ctx, apiClient, input, currentUser, suffix, output)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list pull requests by author: %w", err)
	}

	candidates := make([]approvalCandidate, 0, len(prs))
	for _, pr := range prs {
		candidates = append(candidates, newApprovalCandidate(pr, input.Workspace, ""))
	}
	return candidates, nil
}

func listAuthor(
	ctx context.Context,
	apiClient *client,
	input *URLParams,
	currentUser User,
	author string,
	output io.Writer,
) ([]PullRequest, error) {
	authorInput := *input
	authorInput.Filters = input.Filters
	authorInput.Filters.Author = author

	rawURL, err := buildPullRequestsByAuthorURL(&authorInput, currentUser)
	if err != nil {
		return nil, err
	}
	prs, err := getAllPages[PullRequest](ctx, apiClient, rawURL)
	if !isAPIStatus(err, http.StatusBadRequest) || input.Filters.Project == "" {
		return prs, err
	}

	fmt.Fprintln(
		output,
		"Author query rejected the project predicate; retrying with client-side project filtering.",
	)
	rawURL, buildErr := buildPullRequestsByAuthorURLWithProject(
		&authorInput,
		currentUser,
		false,
	)
	if buildErr != nil {
		return nil, buildErr
	}
	return getAllPages[PullRequest](ctx, apiClient, rawURL)
}
