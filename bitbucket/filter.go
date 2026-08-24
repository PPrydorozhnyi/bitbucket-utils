package bitbucket

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
)

const (
	pullRequestPageLength = 50
	repositoryPageLength  = 100

	pullRequestFields = "next," +
		"values.id,values.title,values.state,values.draft,values.queued," +
		"values.links.html.href,values.links.approve.href,values.destination.repository.slug," +
		"values.destination.repository.full_name,values.destination.repository.project.uuid," +
		"values.reviewers.account_id,values.reviewers.uuid," +
		"values.participants.user.account_id,values.participants.user.uuid," +
		"values.participants.approved,values.author.account_id,values.author.uuid," +
		"values.description"
	repositoryFields = "next,values.slug,values.full_name,values.project.uuid"
)

func buildPullRequestQuery(filters DashboardFilters, includeProject bool) url.Values {
	query := url.Values{
		"fields":  {pullRequestFields},
		"pagelen": {strconv.Itoa(pullRequestPageLength)},
	}
	if filters.Page > 0 {
		query.Set("page", strconv.Itoa(filters.Page))
	}
	if filter := buildPullRequestFilter(filters, includeProject); filter != "" {
		query.Set("q", filter)
	}
	return query
}

func buildRepositoryQuery(project string, page int) url.Values {
	query := url.Values{
		"fields":  {repositoryFields},
		"pagelen": {strconv.Itoa(repositoryPageLength)},
	}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if project = strings.TrimSpace(project); project != "" {
		query.Set("q", "project.uuid = "+quoteQueryValue(project))
	}
	return query
}

func buildPullRequestFilter(filters DashboardFilters, includeProject bool) string {
	clauses := make([]string, 0, 3)
	if state := buildStateFilter(filters.States); state != "" {
		clauses = append(clauses, state)
	}
	if includeProject && filters.Project != "" {
		clauses = append(
			clauses,
			"destination.repository.project.uuid = "+quoteQueryValue(filters.Project),
		)
	}
	if filters.Query != "" {
		clauses = append(clauses, "title ~ "+quoteQueryValue(filters.Query))
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

func quoteQueryValue(value string) string {
	return strconv.Quote(value)
}
