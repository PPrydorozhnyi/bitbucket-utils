package bitbucket

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildRepositoryQuery(t *testing.T) {
	t.Parallel()

	got := buildRepositoryQuery("{project-uuid}", 4)
	assertQueryValue(t, got, "pagelen", "100")
	assertQueryValue(t, got, "page", "4")
	assertQueryValue(t, got, "fields", repositoryFields)
	assertQueryValue(t, got, "q", `project.uuid = "{project-uuid}"`)

	got = buildRepositoryQuery("", 0)
	if got.Has("page") {
		t.Errorf("buildRepositoryQuery() page = %q, want omitted", got.Get("page"))
	}
	if got.Has("q") {
		t.Errorf("buildRepositoryQuery() q = %q, want omitted", got.Get("q"))
	}
}

func TestBuildWorkspaceRepositoriesURL(t *testing.T) {
	t.Parallel()

	raw, err := buildWorkspaceRepositoriesURL("team", "{project-uuid}", 3)
	if err != nil {
		t.Fatalf("buildWorkspaceRepositoriesURL() error = %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Path != "/2.0/repositories/team" {
		t.Errorf("buildWorkspaceRepositoriesURL() path = %q", parsed.Path)
	}
	assertQueryValue(t, parsed.Query(), "page", "3")
	assertQueryValue(t, parsed.Query(), "q", `project.uuid = "{project-uuid}"`)
}

func TestBuildRepositoryPullRequestsURL(t *testing.T) {
	t.Parallel()

	raw, err := buildRepositoryPullRequestsURL(
		"team",
		"repository",
		DashboardFilters{
			States:  []string{"OPEN"},
			Project: "{project-uuid}",
			Page:    2,
		},
		User{AccountID: "current-user"},
	)
	if err != nil {
		t.Fatalf("buildRepositoryPullRequestsURL() error = %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Path != "/2.0/repositories/team/repository/pullrequests" {
		t.Errorf("buildRepositoryPullRequestsURL() path = %q", parsed.Path)
	}
	assertQueryValue(t, parsed.Query(), "page", "2")
	assertQueryValue(
		t,
		parsed.Query(),
		"q",
		`state = "OPEN" AND author.account_id != "current-user"`,
	)
	if strings.Contains(parsed.Query().Get("q"), "project.uuid") {
		t.Errorf("per-repository q unexpectedly contains project: %q", parsed.Query().Get("q"))
	}
}

func TestBuildPullRequestApprovalURL(t *testing.T) {
	t.Parallel()

	raw, err := buildPullRequestApprovalURL("team", "repository", 42)
	if err != nil {
		t.Fatalf("buildPullRequestApprovalURL() error = %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Path != "/2.0/repositories/team/repository/pullrequests/42/approve" {
		t.Errorf("buildPullRequestApprovalURL() path = %q", parsed.Path)
	}

	if _, err := buildPullRequestApprovalURL("team", "", 42); err == nil {
		t.Error("buildPullRequestApprovalURL() accepted an empty repository")
	}
}
