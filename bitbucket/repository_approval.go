package bitbucket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
)

func listSinglePullRequest(
	ctx context.Context,
	apiClient *client,
	input *URLParams,
) ([]approvalCandidate, error) {
	rawURL, err := buildSinglePullRequestURL(input)
	if err != nil {
		return nil, fmt.Errorf("build single pull request URL: %w", err)
	}
	pr, err := getSingle[PullRequest](ctx, apiClient, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch pull request: %w", err)
	}

	return []approvalCandidate{newApprovalCandidate(*pr, input.Workspace, input.Repository)}, nil
}

func newApprovalCandidate(
	pr PullRequest,
	workspace string,
	repository string,
) approvalCandidate {
	if pr.Destination.Repository.Slug != "" {
		repository = pr.Destination.Repository.Slug
	}
	return approvalCandidate{
		PullRequest: pr,
		Workspace:   workspace,
		Repository:  repository,
	}
}

func processCandidates(
	ctx context.Context,
	apiClient *client,
	candidates []approvalCandidate,
	currentUser User,
	dryRun bool,
	output io.Writer,
) (ApprovalSummary, error) {
	summary := ApprovalSummary{Listed: len(candidates)}
	if dryRun {
		for _, candidate := range candidates {
			fmt.Fprintf(output, "Dry run: would approve %s.\n", formatCandidate(candidate))
		}
		return summary, nil
	}

	for _, candidate := range candidates {
		if reason := approvalSkipReason(candidate.PullRequest, currentUser); reason != "" {
			summary.Skipped++
			fmt.Fprintf(output, "Skipped %s: %s.\n", formatCandidate(candidate), reason)
			continue
		}

		approvalURL, err := buildPullRequestApprovalURL(
			candidate.Workspace,
			candidate.Repository,
			candidate.PullRequest.ID,
		)
		if err != nil {
			summary.Failed++
			fmt.Fprintf(output, "Failed %s: %v.\n", formatCandidate(candidate), err)
			continue
		}

		err = apiClient.doJSON(ctx, http.MethodPost, approvalURL, nil)
		switch {
		case err == nil:
			summary.Approved++
			fmt.Fprintf(output, "Approved %s.\n", formatCandidate(candidate))
		case isAPIStatus(err, http.StatusConflict):
			summary.Skipped++
			fmt.Fprintf(
				output,
				"Skipped %s: Bitbucket reports it is already approved.\n",
				formatCandidate(candidate),
			)
		default:
			summary.Failed++
			fmt.Fprintf(output, "Failed %s: %v.\n", formatCandidate(candidate), err)
		}
	}

	if summary.Failed > 0 {
		return summary, fmt.Errorf("%d pull request approval(s) failed", summary.Failed)
	}
	return summary, nil
}

func approvalSkipReason(pr PullRequest, currentUser User) string {
	if sameUser(pr.Author, currentUser) {
		return "authored by current user"
	}
	if slices.ContainsFunc(pr.Participants, func(participant Participant) bool {
		return participant.Approved && sameUser(participant.User, currentUser)
	}) {
		return "already approved by current user"
	}
	return ""
}

func formatCandidate(candidate approvalCandidate) string {
	label := fmt.Sprintf(
		"pull request #%d %q in %s/%s",
		candidate.PullRequest.ID,
		candidate.PullRequest.Title,
		candidate.Workspace,
		candidate.Repository,
	)
	if candidate.PullRequest.Links.HTML.Href != "" {
		label += " (" + candidate.PullRequest.Links.HTML.Href + ")"
	}
	return label
}
