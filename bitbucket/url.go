package bitbucket

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type URLMode string

const (
	URLModeDashboard URLMode = "dashboard"
	URLModeSinglePR  URLMode = "single_pr"

	workspacesAPIURL   = "https://api.bitbucket.org/2.0/workspaces"
	repositoriesAPIURL = "https://api.bitbucket.org/2.0/repositories"
	currentUserURL     = "https://api.bitbucket.org/2.0/user?fields=account_id"
)

var (
	knownQueryParams = map[string]struct{}{
		"author":      {},
		"page":        {},
		"project":     {},
		"query":       {},
		"reviewer":    {},
		"state":       {},
		"user_filter": {},
	}
	validStates = []string{
		"OPEN",
		"DRAFT",
		"QUEUED",
		"MERGED",
		"DECLINED",
		"SUPERSEDED",
	}
	validUserFilters = []string{
		"ALL",
		"AUTHOR",
		"REVIEWING",
		"PARTICIPATING",
	}
)

type URLParams struct {
	Mode          URLMode
	Workspace     string
	Repository    string
	PullRequestID int
	Filters       DashboardFilters
	UnknownParams []string
}

type DashboardFilters struct {
	Author     string
	States     []string
	Project    string
	Query      string
	UserFilter string
	Reviewer   string
	Page       int
}

func parseURL(raw string) (*URLParams, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("URL is required")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme must be https")
	}
	if !strings.EqualFold(parsed.Host, "bitbucket.org") {
		return nil, fmt.Errorf("URL host must be bitbucket.org")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("URL must not contain user information")
	}

	segments := splitPath(parsed.Path)
	switch {
	case isDashboardPath(segments):
		filters, unknown, err := parseDashboardQuery(parsed.RawQuery)
		if err != nil {
			return nil, err
		}
		return &URLParams{
			Mode:          URLModeDashboard,
			Workspace:     segments[0],
			Filters:       filters,
			UnknownParams: unknown,
		}, nil

	case isSinglePRPath(segments):
		id, err := strconv.Atoi(segments[3])
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("pull request ID must be a positive integer")
		}
		return &URLParams{
			Mode:          URLModeSinglePR,
			Workspace:     segments[0],
			Repository:    segments[1],
			PullRequestID: id,
		}, nil

	default:
		return nil, fmt.Errorf(
			"unsupported Bitbucket URL path; expected /{workspace}/workspace/pull-requests/ or /{workspace}/{repo}/pull-requests/{id}",
		)
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return slices.Collect(strings.SplitSeq(trimmed, "/"))
}

func isDashboardPath(segments []string) bool {
	return len(segments) == 3 &&
		segments[0] != "" &&
		segments[1] == "workspace" &&
		segments[2] == "pull-requests"
}

func isSinglePRPath(segments []string) bool {
	return len(segments) == 4 &&
		segments[0] != "" &&
		segments[1] != "" &&
		segments[2] == "pull-requests"
}

func parseDashboardQuery(rawQuery string) (DashboardFilters, []string, error) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return DashboardFilters{}, nil, fmt.Errorf("parse query parameters: %w", err)
	}

	author, err := singleQueryValue(values, "author")
	if err != nil {
		return DashboardFilters{}, nil, err
	}
	project, err := singleQueryValue(values, "project")
	if err != nil {
		return DashboardFilters{}, nil, err
	}
	query, err := singleQueryValue(values, "query")
	if err != nil {
		return DashboardFilters{}, nil, err
	}
	reviewer, err := singleQueryValue(values, "reviewer")
	if err != nil {
		return DashboardFilters{}, nil, err
	}
	pageValue, err := singleQueryValue(values, "page")
	if err != nil {
		return DashboardFilters{}, nil, err
	}
	userFilter, err := singleQueryValue(values, "user_filter")
	if err != nil {
		return DashboardFilters{}, nil, err
	}

	states, err := parseStates(values["state"])
	if err != nil {
		return DashboardFilters{}, nil, err
	}

	userFilter = strings.ToUpper(strings.TrimSpace(userFilter))
	if userFilter == "" {
		userFilter = "REVIEWING"
	}
	if !slices.Contains(validUserFilters, userFilter) {
		return DashboardFilters{}, nil, fmt.Errorf(
			"unsupported user_filter %q; expected one of %s",
			userFilter,
			strings.Join(validUserFilters, ", "),
		)
	}

	page := 0
	if _, provided := values["page"]; provided {
		page, err = strconv.Atoi(strings.TrimSpace(pageValue))
		if err != nil || page <= 0 {
			return DashboardFilters{}, nil, fmt.Errorf("query parameter %q must be a positive integer", "page")
		}
	}

	var unknown []string
	for _, key := range slices.Sorted(maps.Keys(values)) {
		if _, ok := knownQueryParams[key]; !ok {
			unknown = append(unknown, key)
		}
	}

	return DashboardFilters{
		Author:     strings.TrimSpace(author),
		States:     states,
		Project:    strings.TrimSpace(project),
		Query:      query,
		UserFilter: userFilter,
		Reviewer:   strings.TrimSpace(reviewer),
		Page:       page,
	}, unknown, nil
}

func singleQueryValue(values url.Values, key string) (string, error) {
	items, ok := values[key]
	if !ok {
		return "", nil
	}
	if len(items) != 1 {
		return "", fmt.Errorf("query parameter %q must appear at most once", key)
	}
	return items[0], nil
}

func parseStates(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"OPEN"}, nil
	}

	states := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		tokens := strings.FieldsFunc(value, func(r rune) bool {
			return r == '+' || unicode.IsSpace(r)
		})
		if len(tokens) == 0 {
			return nil, fmt.Errorf("query parameter %q must not be empty", "state")
		}
		for _, token := range tokens {
			state := strings.ToUpper(token)
			if !slices.Contains(validStates, state) {
				return nil, fmt.Errorf(
					"unsupported state %q; expected one of %s",
					token,
					strings.Join(validStates, ", "),
				)
			}
			if _, ok := seen[state]; ok {
				continue
			}
			seen[state] = struct{}{}
			states = append(states, state)
		}
	}

	return states, nil
}

func buildPullRequestsByAuthorURL(input *URLParams, currentUser User) (string, error) {
	u, err := url.Parse(workspacesAPIURL)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(input.Workspace, "pullrequests", input.Filters.Author)
	u.RawQuery = buildDashboardPullRequestQuery(input.Filters, currentUser).Encode()
	return u.String(), nil
}

func buildWorkspaceRepositoriesURL(workspace, project string, page int) (string, error) {
	u, err := url.Parse(repositoriesAPIURL)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(workspace)
	u.RawQuery = buildRepositoryQuery(project, page).Encode()
	return u.String(), nil
}

func buildRepositoryPullRequestsURL(
	workspace string,
	repository string,
	filters DashboardFilters,
	currentUser User,
) (string, error) {
	u, err := url.Parse(repositoriesAPIURL)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(workspace, repository, "pullrequests")
	u.RawQuery = buildRepositoryPullRequestQuery(filters, currentUser).Encode()
	return u.String(), nil
}
