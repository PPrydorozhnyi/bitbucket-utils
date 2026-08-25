package bitbucket

import (
	"net/url"
	"strconv"
	"strings"
)

const (
	repositoryPageLength = 100
	repositoryFields     = "next,values.slug,values.full_name,values.project.uuid"
)

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

func buildRepositoryPullRequestQuery(filters DashboardFilters, currentUser User) url.Values {
	return buildPullRequestQuery(filters, pullRequestQueryOptions{
		ExcludeAuthorAccountID: currentUser.AccountID,
	})
}
