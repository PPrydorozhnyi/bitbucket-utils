package bitbucket

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunApprovalAuthorFallbacks(t *testing.T) {
	t.Parallel()

	var originalAuthorRequests, suffixRequests int
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assertBasicAuth(t, request)
		switch request.URL.Path {
		case "/2.0/workspaces/team/pullrequests/prefix:suffix":
			originalAuthorRequests++
			writeAPIError(response, http.StatusNotFound, "Author not found")
		case "/2.0/workspaces/team/pullrequests/suffix":
			suffixRequests++
			if strings.Contains(request.URL.Query().Get("q"), "project.uuid") {
				writeAPIError(response, http.StatusBadRequest, "Nested project query unsupported")
				return
			}
			if request.URL.Query().Get("page") == "2" {
				fmt.Fprint(response, `{"values":[{
					"id":2,
					"title":"Other project",
					"state":"OPEN",
					"destination":{"repository":{
						"slug":"two",
						"full_name":"team/two",
						"project":{"uuid":"{other-project}"}
					}}
				}]}`)
				return
			}
			fmt.Fprint(response, `{"values":[{
				"id":1,
				"title":"Matching project",
				"state":"OPEN",
				"destination":{"repository":{
					"slug":"one",
					"full_name":"team/one",
					"project":{"uuid":"{project-uuid}"}
				}}
			}],"next":"?page=2"}`)
		default:
			http.Error(response, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	summary, err := runApproval(
		t.Context(),
		"https://bitbucket.org/team/workspace/pull-requests/"+
			"?author=prefix%3Asuffix&project=%7Bproject-uuid%7D&state=OPEN&user_filter=ALL",
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
		t.Errorf("summary = %+v, want one filtered pull request", summary)
	}
	if originalAuthorRequests != 1 || suffixRequests != 3 {
		t.Errorf(
			"author request counts = original:%d suffix:%d, want 1 and 3",
			originalAuthorRequests,
			suffixRequests,
		)
	}
	for _, text := range []string{
		"retrying with UUID suffix",
		"retrying with client-side project filtering",
	} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("output %q does not contain %q", output.String(), text)
		}
	}
}
