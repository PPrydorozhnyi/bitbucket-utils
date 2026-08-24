package bitbucket

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseURLDashboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want *URLParams
	}{
		{
			name: "full dashboard URL",
			raw: "https://bitbucket.org/delasport/workspace/pull-requests/?" +
				"author=712020%3A47ee3e5f-1b5c-4616-9f46-77e67b1894cf" +
				"&state=OPEN%2BDRAFT%2BQUEUED" +
				"&user_filter=ALL" +
				"&project=%7Ba747af2d-ade5-4e49-92c1-b6bc6480a16a%7D" +
				"&query=jacoco",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "delasport",
				Filters: DashboardFilters{
					Author:     "712020:47ee3e5f-1b5c-4616-9f46-77e67b1894cf",
					States:     []string{"OPEN", "DRAFT", "QUEUED"},
					Project:    "{a747af2d-ade5-4e49-92c1-b6bc6480a16a}",
					Query:      "jacoco",
					UserFilter: "ALL",
				},
			},
		},
		{
			name: "no author with project",
			raw:  "https://bitbucket.org/team/workspace/pull-requests/?project=%7Bproject-uuid%7D&state=OPEN",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "team",
				Filters: DashboardFilters{
					States:     []string{"OPEN"},
					Project:    "{project-uuid}",
					UserFilter: "ALL",
				},
			},
		},
		{
			name: "no author or project",
			raw:  "https://bitbucket.org/team/workspace/pull-requests/",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "team",
				Filters: DashboardFilters{
					States:     []string{"OPEN"},
					UserFilter: "ALL",
				},
			},
		},
		{
			name: "literal plus state separator",
			raw:  "https://bitbucket.org/team/workspace/pull-requests/?state=OPEN+DRAFT+QUEUED",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "team",
				Filters: DashboardFilters{
					States:     []string{"OPEN", "DRAFT", "QUEUED"},
					UserFilter: "ALL",
				},
			},
		},
		{
			name: "repeated states are combined and deduplicated",
			raw:  "https://bitbucket.org/team/workspace/pull-requests/?state=open%2Bdraft&state=DRAFT&state=merged",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "team",
				Filters: DashboardFilters{
					States:     []string{"OPEN", "DRAFT", "MERGED"},
					UserFilter: "ALL",
				},
			},
		},
		{
			name: "query and role filters",
			raw:  "https://bitbucket.org/team/workspace/pull-requests/?query=hello+world%22&reviewer=abc&user_filter=reviewing",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "team",
				Filters: DashboardFilters{
					States:     []string{"OPEN"},
					Query:      "hello world\"",
					UserFilter: "REVIEWING",
					Reviewer:   "abc",
				},
			},
		},
		{
			name: "unknown parameters are sorted",
			raw:  "https://bitbucket.org/team/workspace/pull-requests/?z=1&state=OPEN&a=2",
			want: &URLParams{
				Mode:      URLModeDashboard,
				Workspace: "team",
				Filters: DashboardFilters{
					States:     []string{"OPEN"},
					UserFilter: "ALL",
				},
				UnknownParams: []string{"a", "z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseURL(tt.raw)
			if err != nil {
				t.Fatalf("parseURL() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseURL() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseURLSinglePR(t *testing.T) {
	t.Parallel()

	got, err := parseURL(
		"https://BITBUCKET.ORG/delasport/repository-name/pull-requests/42/?ignored=1",
	)
	if err != nil {
		t.Fatalf("parseURL() error = %v", err)
	}

	want := &URLParams{
		Mode:          URLModeSinglePR,
		Workspace:     "delasport",
		Repository:    "repository-name",
		PullRequestID: 42,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseURL() = %#v, want %#v", got, want)
	}
}

func TestParseURLRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "empty", raw: "", wantErr: "URL is required"},
		{name: "wrong scheme", raw: "http://bitbucket.org/team/workspace/pull-requests/", wantErr: "scheme must be https"},
		{name: "wrong host", raw: "https://example.com/team/workspace/pull-requests/", wantErr: "host must be bitbucket.org"},
		{name: "host with port", raw: "https://bitbucket.org:443/team/workspace/pull-requests/", wantErr: "host must be bitbucket.org"},
		{name: "user information", raw: "https://user@bitbucket.org/team/workspace/pull-requests/", wantErr: "must not contain user information"},
		{name: "wrong path", raw: "https://bitbucket.org/team/repository", wantErr: "unsupported Bitbucket URL path"},
		{name: "zero pull request ID", raw: "https://bitbucket.org/team/repo/pull-requests/0", wantErr: "positive integer"},
		{name: "invalid pull request ID", raw: "https://bitbucket.org/team/repo/pull-requests/latest", wantErr: "positive integer"},
		{name: "empty state", raw: "https://bitbucket.org/team/workspace/pull-requests/?state=", wantErr: `"state" must not be empty`},
		{name: "unsupported state", raw: "https://bitbucket.org/team/workspace/pull-requests/?state=CLOSED", wantErr: `unsupported state "CLOSED"`},
		{name: "unsupported user filter", raw: "https://bitbucket.org/team/workspace/pull-requests/?user_filter=MINE", wantErr: `unsupported user_filter "MINE"`},
		{name: "duplicate singleton", raw: "https://bitbucket.org/team/workspace/pull-requests/?author=a&author=b", wantErr: `"author" must appear at most once`},
		{name: "malformed query escape", raw: "https://bitbucket.org/team/workspace/pull-requests/?query=%zz", wantErr: "invalid URL escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseURL(tt.raw)
			if err == nil {
				t.Fatal("parseURL() error = nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseURL() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
