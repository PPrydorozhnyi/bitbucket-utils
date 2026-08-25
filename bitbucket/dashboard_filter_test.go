package bitbucket

import (
	"net/url"
	"reflect"
	"testing"
)

func TestBuildStateFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		states []string
		want   string
	}{
		{name: "open", states: []string{"OPEN"}, want: `state = "OPEN"`},
		{
			name:   "draft includes open and draft visibility clause",
			states: []string{"DRAFT"},
			want:   `(state = "OPEN" AND (draft = true OR draft = false))`,
		},
		{
			name:   "queued includes open and draft visibility clause",
			states: []string{"QUEUED"},
			want:   `(state = "OPEN" AND (draft = true OR draft = false))`,
		},
		{
			name:   "open draft queued share one open clause",
			states: []string{"OPEN", "DRAFT", "QUEUED"},
			want:   `(state = "OPEN" AND (draft = true OR draft = false))`,
		},
		{
			name:   "multiple classic states use or",
			states: []string{"MERGED", "DECLINED", "SUPERSEDED"},
			want:   `(state = "MERGED" OR state = "DECLINED" OR state = "SUPERSEDED")`,
		},
		{
			name:   "draft and merged use grouped or",
			states: []string{"DRAFT", "MERGED"},
			want:   `((state = "OPEN" AND (draft = true OR draft = false)) OR state = "MERGED")`,
		},
		{name: "empty", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildStateFilter(tt.states); got != tt.want {
				t.Errorf("buildStateFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPullRequestFilter(t *testing.T) {
	t.Parallel()

	filters := DashboardFilters{
		States:  []string{"OPEN", "DRAFT"},
		Project: "{project-uuid}",
		Query:   `say "hi" \ path`,
	}
	got := buildPullRequestFilter(filters, pullRequestQueryOptions{
		IncludeProject:         true,
		ExcludeAuthorAccountID: "current-user",
	})
	want := `(state = "OPEN" AND (draft = true OR draft = false))` +
		` AND destination.repository.project.uuid = "{project-uuid}"` +
		` AND title ~ "say \"hi\" \\ path"` +
		` AND author.account_id != "current-user"`
	if got != want {
		t.Errorf("buildPullRequestFilter() = %q, want %q", got, want)
	}

	got = buildPullRequestFilter(filters, pullRequestQueryOptions{})
	want = `(state = "OPEN" AND (draft = true OR draft = false))` +
		` AND title ~ "say \"hi\" \\ path"`
	if got != want {
		t.Errorf("buildPullRequestFilter() without project = %q, want %q", got, want)
	}
}

func TestBuildDashboardPullRequestQuery(t *testing.T) {
	t.Parallel()

	got := buildDashboardPullRequestQuery(DashboardFilters{
		States:  []string{"OPEN"},
		Project: "{project-uuid}",
		Query:   "jacoco",
		Page:    3,
	}, User{AccountID: "current-user"})

	assertQueryValue(t, got, "pagelen", "50")
	assertQueryValue(t, got, "page", "3")
	assertQueryValue(t, got, "fields", pullRequestFields)
	assertQueryValue(
		t,
		got,
		"q",
		`state = "OPEN" AND destination.repository.project.uuid = "{project-uuid}"`+
			` AND title ~ "jacoco" AND author.account_id != "current-user"`,
	)
}

func TestBuildPullRequestsByAuthorURL(t *testing.T) {
	t.Parallel()

	input, err := parseURL(
		"https://bitbucket.org/team/workspace/pull-requests/?" +
			"author=user-id&state=OPEN%2BDRAFT&project=%7Bproject-uuid%7D&query=jacoco&page=2",
	)
	if err != nil {
		t.Fatalf("parseURL() error = %v", err)
	}

	raw, err := buildPullRequestsByAuthorURL(input, User{AccountID: "current-user"})
	if err != nil {
		t.Fatalf("buildPullRequestsByAuthorURL() error = %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Path != "/2.0/workspaces/team/pullrequests/user-id" {
		t.Errorf("buildPullRequestsByAuthorURL() path = %q", parsed.Path)
	}
	assertQueryValue(t, parsed.Query(), "page", "2")
	assertQueryValue(
		t,
		parsed.Query(),
		"q",
		`(state = "OPEN" AND (draft = true OR draft = false))`+
			` AND destination.repository.project.uuid = "{project-uuid}"`+
			` AND title ~ "jacoco"`+
			` AND author.account_id != "current-user"`,
	)
}

func TestFilterApprovablePRs(t *testing.T) {
	t.Parallel()

	currentUser := User{AccountID: "current-user"}
	prs := []PullRequest{
		{Id: 1, Author: currentUser},
		{
			Id:           2,
			Author:       User{AccountID: "other"},
			Participants: []Participant{{User: currentUser, Approved: true}},
		},
		{
			Id:           3,
			Author:       User{AccountID: "other"},
			Participants: []Participant{{User: currentUser}},
		},
		{
			Id:     4,
			Author: User{AccountID: "other"},
			Participants: []Participant{{
				User:     User{AccountID: "another-user"},
				Approved: true,
			}},
		},
	}

	got := filterApprovablePRs(prs, currentUser)
	want := []PullRequest{prs[2], prs[3]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filterApprovablePRs() = %#v, want %#v", got, want)
	}
	if len(prs) != 4 {
		t.Errorf("filterApprovablePRs() mutated input length to %d", len(prs))
	}
}

func assertQueryValue(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Errorf("query parameter %q = %q, want %q", key, got, want)
	}
}
