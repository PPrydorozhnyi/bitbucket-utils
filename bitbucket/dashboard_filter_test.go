package bitbucket

import (
	"net/url"
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

func TestBuildPullRequestQuery(t *testing.T) {
	t.Parallel()

	got := buildPullRequestQuery(
		DashboardFilters{
			States:  []string{"OPEN"},
			Project: "{project-uuid}",
			Query:   "jacoco",
			Page:    3,
		},
		pullRequestQueryOptions{
			IncludeProject:         true,
			ExcludeAuthorAccountID: "current-user",
		},
	)

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

func TestMatchesDashboardFilters(t *testing.T) {
	t.Parallel()

	currentUser := User{AccountID: "current-user", UUID: "{current-uuid}"}
	base := PullRequest{
		ID:          1,
		Title:       "Improve coverage",
		Description: "Adds JaCoCo reporting",
		State:       "OPEN",
		Destination: Destination{
			Repository: Repository{Project: Project{UUID: "{project-uuid}"}},
		},
		Reviewers: []User{
			currentUser,
			{AccountID: "selected-reviewer", UUID: "{reviewer-uuid}"},
		},
		Participants: []Participant{{User: currentUser}},
	}

	tests := []struct {
		name    string
		change  func(*PullRequest)
		filters DashboardFilters
		want    bool
	}{
		{
			name:    "open",
			filters: DashboardFilters{States: []string{"OPEN"}, UserFilter: "ALL"},
			want:    true,
		},
		{
			name:   "open excludes draft",
			change: func(pr *PullRequest) { pr.Draft = true },
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				UserFilter: "ALL",
			},
		},
		{
			name:   "draft",
			change: func(pr *PullRequest) { pr.Draft = true },
			filters: DashboardFilters{
				States:     []string{"DRAFT"},
				UserFilter: "ALL",
			},
			want: true,
		},
		{
			name:   "queued",
			change: func(pr *PullRequest) { pr.Queued = true },
			filters: DashboardFilters{
				States:     []string{"QUEUED"},
				UserFilter: "ALL",
			},
			want: true,
		},
		{
			name:   "classic state",
			change: func(pr *PullRequest) { pr.State = "MERGED" },
			filters: DashboardFilters{
				States:     []string{"MERGED"},
				UserFilter: "ALL",
			},
			want: true,
		},
		{
			name: "project mismatch",
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				Project:    "{other-project}",
				UserFilter: "ALL",
			},
		},
		{
			name: "description query is case insensitive",
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				Query:      "JACOCO",
				UserFilter: "ALL",
			},
			want: true,
		},
		{
			name: "reviewer UUID",
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				Reviewer:   "{REVIEWER-UUID}",
				UserFilter: "ALL",
			},
			want: true,
		},
		{
			name: "reviewing",
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				UserFilter: "REVIEWING",
			},
			want: true,
		},
		{
			name: "participating",
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				UserFilter: "PARTICIPATING",
			},
			want: true,
		},
		{
			name: "query mismatch",
			filters: DashboardFilters{
				States:     []string{"OPEN"},
				Query:      "unrelated",
				UserFilter: "ALL",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pr := base
			if tt.change != nil {
				tt.change(&pr)
			}
			if got := matchesDashboardFilters(pr, tt.filters, currentUser); got != tt.want {
				t.Errorf("matchesDashboardFilters() = %t, want %t", got, tt.want)
			}
		})
	}
}

func assertQueryValue(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Errorf("query parameter %q = %q, want %q", key, got, want)
	}
}
