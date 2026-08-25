package bitbucket

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
)

const (
	pullRequestPageLength = 50
	pullRequestFields     = "next," +
		"values.id,values.title,values.state,values.draft,values.queued," +
		"values.links.html.href,values.links.approve.href,values.destination.repository.slug," +
		"values.destination.repository.full_name,values.destination.repository.project.uuid," +
		"values.reviewers.account_id,values.reviewers.uuid," +
		"values.participants.user.account_id,values.participants.user.uuid," +
		"values.participants.approved,values.author.account_id,values.author.uuid," +
		"values.description"
)

type pullRequestQueryOptions struct {
	IncludeProject         bool
	ExcludeAuthorAccountID string
}

func buildDashboardPullRequestQuery(filters DashboardFilters, currentUser User) url.Values {
	return buildPullRequestQuery(filters, pullRequestQueryOptions{
		IncludeProject:         true,
		ExcludeAuthorAccountID: currentUser.AccountID,
	})
}

func buildPullRequestQuery(filters DashboardFilters, options pullRequestQueryOptions) url.Values {
	query := url.Values{
		"fields":  {pullRequestFields},
		"pagelen": {strconv.Itoa(pullRequestPageLength)},
	}
	if filters.Page > 0 {
		query.Set("page", strconv.Itoa(filters.Page))
	}
	if filter := buildPullRequestFilter(filters, options); filter != "" {
		query.Set("q", filter)
	}
	return query
}

func buildPullRequestFilter(filters DashboardFilters, options pullRequestQueryOptions) string {
	clauses := make([]string, 0, 4)
	if state := buildStateFilter(filters.States); state != "" {
		clauses = append(clauses, state)
	}
	if options.IncludeProject && filters.Project != "" {
		clauses = append(
			clauses,
			"destination.repository.project.uuid = "+quoteQueryValue(filters.Project),
		)
	}
	if filters.Query != "" {
		clauses = append(clauses, "title ~ "+quoteQueryValue(filters.Query))
	}
	if options.ExcludeAuthorAccountID != "" {
		clauses = append(
			clauses,
			"author.account_id != "+quoteQueryValue(options.ExcludeAuthorAccountID),
		)
	}
	return strings.Join(clauses, " AND ")
}

func buildStateFilter(states []string) string {
	classicStates := make([]string, 0, len(states))
	includeOpenFlags := slices.Contains(states, "DRAFT") || slices.Contains(states, "QUEUED")

	for _, state := range states {
		switch state {
		case "OPEN":
			if !slices.Contains(classicStates, state) {
				classicStates = append(classicStates, state)
			}
		case "MERGED", "DECLINED", "SUPERSEDED":
			if !slices.Contains(classicStates, state) {
				classicStates = append(classicStates, state)
			}
		}
	}
	if includeOpenFlags && !slices.Contains(classicStates, "OPEN") {
		classicStates = append([]string{"OPEN"}, classicStates...)
	}

	clauses := make([]string, 0, len(classicStates))
	for _, state := range classicStates {
		clause := "state = " + quoteQueryValue(state)
		if state == "OPEN" && includeOpenFlags {
			clause = "(" + clause + " AND (draft = true OR draft = false))"
		}
		clauses = append(clauses, clause)
	}

	switch len(clauses) {
	case 0:
		return ""
	case 1:
		return clauses[0]
	default:
		return "(" + strings.Join(clauses, " OR ") + ")"
	}
}

func filterApprovablePRs(prs []PullRequest, currentUser User) []PullRequest {
	return slices.DeleteFunc(slices.Clone(prs), func(pr PullRequest) bool {
		if pr.Author.AccountID == currentUser.AccountID {
			return true
		}
		return slices.ContainsFunc(pr.Participants, func(participant Participant) bool {
			return participant.Approved &&
				participant.User.AccountID == currentUser.AccountID
		})
	})
}

func quoteQueryValue(value string) string {
	return strconv.Quote(value)
}
