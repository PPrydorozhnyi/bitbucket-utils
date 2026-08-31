package bitbucket

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

type ApprovalOptions struct {
	DryRun bool
	Output io.Writer
}

type ApprovalSummary struct {
	Listed   int
	Approved int
	Skipped  int
	Failed   int
}

type approvalRunOptions struct {
	DryRun    bool
	Output    io.Writer
	APIClient *client
}

type approvalCandidate struct {
	PullRequest PullRequest
	Workspace   string
	Repository  string
}

func ApproveWithOptions(u string, options ApprovalOptions) (ApprovalSummary, error) {
	return runApproval(context.Background(), u, approvalRunOptions{
		DryRun: options.DryRun,
		Output: options.Output,
	})
}

func runApproval(
	ctx context.Context,
	rawURL string,
	options approvalRunOptions,
) (ApprovalSummary, error) {
	if options.Output == nil {
		options.Output = os.Stdout
	}

	input, err := parseURL(rawURL)
	if err != nil {
		return ApprovalSummary{}, err
	}
	printInput(options.Output, input, options.DryRun)
	for _, key := range input.UnknownParams {
		fmt.Fprintf(
			options.Output,
			"Warning: unsupported query parameter %q was not applied\n",
			key,
		)
	}

	apiClient := options.APIClient
	if apiClient == nil {
		credentials, err := getCreds()
		if err != nil {
			return ApprovalSummary{}, err
		}
		apiClient = newClient(nil, credentials)
	}

	if input.Mode == URLModeDashboard &&
		input.Filters.Author == "" &&
		input.Filters.UserFilter == "AUTHOR" {
		fmt.Fprintln(
			options.Output,
			"No approvable pull requests: user_filter=AUTHOR selects your own pull requests.",
		)
		summary := ApprovalSummary{}
		printSummary(options.Output, summary)
		return summary, nil
	}

	var currentUser User
	if needsCurrentUser(input, options.DryRun) {
		user, err := getSingle[User](ctx, apiClient, currentUserURL)
		if err != nil {
			return ApprovalSummary{}, fmt.Errorf("fetch current Bitbucket user: %w", err)
		}
		if user.AccountID == "" {
			return ApprovalSummary{}, fmt.Errorf("current Bitbucket user response has no account_id")
		}
		currentUser = *user
	}

	candidates, err := listCandidates(
		ctx,
		apiClient,
		input,
		currentUser,
		options.Output,
	)
	if err != nil {
		return ApprovalSummary{}, err
	}
	if input.Mode == URLModeDashboard {
		candidates = filterDashboardCandidates(candidates, input.Filters, currentUser)
	}
	candidates = deduplicateCandidates(candidates)

	summary, approvalErr := processCandidates(
		ctx,
		apiClient,
		candidates,
		currentUser,
		options.DryRun,
		options.Output,
	)
	printSummary(options.Output, summary)
	return summary, approvalErr
}

func getCreds() (*Credentials, error) {
	email := strings.TrimSpace(os.Getenv("BITBUCKET_EMAIL"))
	if email == "" {
		return nil, fmt.Errorf("BITBUCKET_EMAIL environment variable is not set")
	}

	token := os.Getenv("BITBUCKET_API_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("BITBUCKET_API_TOKEN environment variable is not set")
	}

	return &Credentials{User: email, Token: token}, nil
}

func needsCurrentUser(input *URLParams, dryRun bool) bool {
	if !dryRun {
		return true
	}
	if input.Mode != URLModeDashboard {
		return false
	}
	return input.Filters.UserFilter == "REVIEWING" ||
		input.Filters.UserFilter == "PARTICIPATING"
}

func listCandidates(
	ctx context.Context,
	apiClient *client,
	input *URLParams,
	currentUser User,
	output io.Writer,
) ([]approvalCandidate, error) {
	switch input.Mode {
	case URLModeSinglePR:
		return listSinglePullRequest(ctx, apiClient, input)
	case URLModeDashboard:
		if input.Filters.Author != "" {
			return listPullRequestsByAuthor(ctx, apiClient, input, currentUser, output)
		}
		return listPullRequestsByRepository(ctx, apiClient, input, currentUser, output)
	default:
		return nil, fmt.Errorf("unsupported Bitbucket URL mode %q", input.Mode)
	}
}

func deduplicateCandidates(candidates []approvalCandidate) []approvalCandidate {
	type candidateKey struct {
		Workspace  string
		Repository string
		ID         int
	}

	seen := make(map[candidateKey]struct{}, len(candidates))
	return slices.DeleteFunc(slices.Clone(candidates), func(candidate approvalCandidate) bool {
		repository := candidate.Repository
		if fullName := candidate.PullRequest.Destination.Repository.FullName; fullName != "" {
			repository = fullName
		}
		key := candidateKey{
			Workspace:  candidate.Workspace,
			Repository: repository,
			ID:         candidate.PullRequest.ID,
		}
		if _, exists := seen[key]; exists {
			return true
		}
		seen[key] = struct{}{}
		return false
	})
}

func printInput(output io.Writer, input *URLParams, dryRun bool) {
	if input.Mode == URLModeSinglePR {
		fmt.Fprintf(
			output,
			"Mode: single pull request; workspace=%q repository=%q id=%d dry_run=%t\n",
			input.Workspace,
			input.Repository,
			input.PullRequestID,
			dryRun,
		)
		return
	}

	fmt.Fprintf(
		output,
		"Mode: dashboard; workspace=%q author=%q states=%q project=%q query=%q user_filter=%q reviewer=%q page=%d dry_run=%t\n",
		input.Workspace,
		input.Filters.Author,
		strings.Join(input.Filters.States, ","),
		input.Filters.Project,
		input.Filters.Query,
		input.Filters.UserFilter,
		input.Filters.Reviewer,
		input.Filters.Page,
		dryRun,
	)
}

func printSummary(output io.Writer, summary ApprovalSummary) {
	fmt.Fprintf(
		output,
		"Summary: listed=%d approved=%d skipped=%d failed=%d\n",
		summary.Listed,
		summary.Approved,
		summary.Skipped,
		summary.Failed,
	)
}
