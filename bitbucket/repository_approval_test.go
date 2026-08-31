package bitbucket

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestRunApprovalSinglePullRequest(t *testing.T) {
	t.Parallel()

	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assertBasicAuth(t, request)
		requests = append(requests, request.Method+" "+request.URL.Path)

		switch request.URL.Path {
		case "/2.0/user":
			fmt.Fprint(response, `{"account_id":"current-user","uuid":"{current-uuid}"}`)
		case "/2.0/repositories/team/repository/pullrequests/42":
			fmt.Fprint(response, `{
				"id":42,
				"title":"Fix race",
				"state":"OPEN",
				"author":{"account_id":"other-user"},
				"destination":{"repository":{"slug":"repository","full_name":"team/repository"}}
			}`)
		case "/2.0/repositories/team/repository/pullrequests/42/approve":
			response.WriteHeader(http.StatusNoContent)
		default:
			http.Error(response, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	summary, err := runApproval(
		t.Context(),
		"https://bitbucket.org/team/repository/pull-requests/42",
		approvalRunOptions{
			Output:    &output,
			APIClient: testAPIClient(t, server),
		},
	)
	if err != nil {
		t.Fatalf("runApproval() error = %v", err)
	}
	wantSummary := ApprovalSummary{Listed: 1, Approved: 1}
	if summary != wantSummary {
		t.Errorf("summary = %+v, want %+v", summary, wantSummary)
	}
	wantRequests := []string{
		"GET /2.0/user",
		"GET /2.0/repositories/team/repository/pullrequests/42",
		"POST /2.0/repositories/team/repository/pullrequests/42/approve",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Errorf("requests = %#v, want %#v", requests, wantRequests)
	}
	if !strings.Contains(output.String(), "Summary: listed=1 approved=1 skipped=0 failed=0") {
		t.Errorf("output = %q", output.String())
	}
}

func TestRunApprovalDryRunDoesNotFetchCurrentUserOrPost(t *testing.T) {
	t.Parallel()

	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assertBasicAuth(t, request)
		requests = append(requests, request.Method+" "+request.URL.Path)

		if request.URL.Path != "/2.0/repositories/team/repository/pullrequests/42" {
			http.Error(response, "unexpected request", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(response, `{
			"id":42,
			"title":"Fix race",
			"state":"OPEN",
			"destination":{"repository":{"slug":"repository"}}
		}`)
	}))
	defer server.Close()

	var output bytes.Buffer
	summary, err := runApproval(
		t.Context(),
		"https://bitbucket.org/team/repository/pull-requests/42",
		approvalRunOptions{
			DryRun:    true,
			Output:    &output,
			APIClient: testAPIClient(t, server),
		},
	)
	if err != nil {
		t.Fatalf("runApproval() error = %v", err)
	}
	if summary != (ApprovalSummary{Listed: 1}) {
		t.Errorf("summary = %+v, want one listed pull request", summary)
	}
	if want := []string{"GET /2.0/repositories/team/repository/pullrequests/42"}; !reflect.DeepEqual(requests, want) {
		t.Errorf("requests = %#v, want %#v", requests, want)
	}
	if !strings.Contains(output.String(), "Dry run: would approve") {
		t.Errorf("output = %q", output.String())
	}
}
