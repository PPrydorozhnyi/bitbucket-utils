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

func TestRunApprovalProjectRepositoryScanAppliesUserRoleFilters(t *testing.T) {
	t.Parallel()

	var repositoryPages int
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assertBasicAuth(t, request)
		switch request.URL.Path {
		case "/2.0/user":
			fmt.Fprint(response, `{"account_id":"current-user","uuid":"{current-uuid}"}`)
		case "/2.0/repositories/team":
			repositoryPages++
			if request.URL.Query().Get("page") == "2" {
				fmt.Fprint(response, `{"values":[{"slug":"two"}]}`)
				return
			}
			if !strings.Contains(request.URL.Query().Get("q"), "project.uuid") {
				t.Errorf("repository query = %q, want project filter", request.URL.RawQuery)
			}
			fmt.Fprint(response, `{"values":[{"slug":"one"}],"next":"?page=2"}`)
		case "/2.0/repositories/team/one/pullrequests":
			fmt.Fprint(response, `{"values":[{
				"id":1,
				"title":"Review me",
				"state":"OPEN",
				"reviewers":[{"account_id":"current-user"}],
				"destination":{"repository":{"slug":"one","project":{"uuid":"{project-uuid}"}}}
			}]}`)
		case "/2.0/repositories/team/two/pullrequests":
			fmt.Fprint(response, `{"values":[{
				"id":2,
				"title":"Participating",
				"state":"OPEN",
				"reviewers":[{"account_id":"other-user"}],
				"participants":[{"user":{"account_id":"current-user"}}],
				"destination":{"repository":{"slug":"two","project":{"uuid":"{project-uuid}"}}}
			}]}`)
		default:
			http.Error(response, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	for _, userFilter := range []string{"REVIEWING", "PARTICIPATING"} {
		t.Run(strings.ToLower(userFilter), func(t *testing.T) {
			repositoryPages = 0
			var output bytes.Buffer
			summary, err := runApproval(
				t.Context(),
				"https://bitbucket.org/team/workspace/pull-requests/"+
					"?project=%7Bproject-uuid%7D&state=OPEN&user_filter="+userFilter,
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
				t.Errorf("summary = %+v, want one %s pull request", summary, userFilter)
			}
			if repositoryPages != 2 {
				t.Errorf("repository page requests = %d, want 2", repositoryPages)
			}
			if !strings.Contains(output.String(), "Scanning 2 project repositories") {
				t.Errorf("output = %q", output.String())
			}
		})
	}
}

func TestRunApprovalContinuesAfterFailuresAndCountsSkips(t *testing.T) {
	t.Parallel()

	postCounts := make(map[int]int)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assertBasicAuth(t, request)
		switch request.URL.Path {
		case "/2.0/user":
			fmt.Fprint(response, `{"account_id":"current-user"}`)
		case "/2.0/repositories/team":
			fmt.Fprint(response, `{"values":[{"slug":"repository"}]}`)
		case "/2.0/repositories/team/repository/pullrequests":
			fmt.Fprint(response, `{"values":[
				{"id":1,"title":"Own","state":"OPEN","author":{"account_id":"current-user"}},
				{"id":2,"title":"Approved","state":"OPEN","author":{"account_id":"other"},
				 "participants":[{"user":{"account_id":"current-user"},"approved":true}]},
				{"id":5,"title":"Forbidden","state":"OPEN","author":{"account_id":"other"}},
				{"id":3,"title":"Success","state":"OPEN","author":{"account_id":"other"}},
				{"id":4,"title":"Conflict","state":"OPEN","author":{"account_id":"other"}},
				{"id":3,"title":"Duplicate","state":"OPEN","author":{"account_id":"other"}}
			]}`)
		default:
			if request.Method != http.MethodPost {
				http.Error(response, "unexpected request", http.StatusInternalServerError)
				return
			}
			id, err := approvalPathID(request.URL.Path)
			if err != nil {
				t.Errorf("approval path: %v", err)
				http.Error(response, "bad path", http.StatusBadRequest)
				return
			}
			postCounts[id]++
			switch id {
			case 3:
				response.WriteHeader(http.StatusNoContent)
			case 4:
				writeAPIError(response, http.StatusConflict, "Already approved")
			case 5:
				writeAPIError(response, http.StatusForbidden, "Missing scope")
			default:
				http.Error(response, "unexpected pull request", http.StatusBadRequest)
			}
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	summary, err := runApproval(
		t.Context(),
		"https://bitbucket.org/team/workspace/pull-requests/?state=OPEN&user_filter=ALL",
		approvalRunOptions{
			Output:    &output,
			APIClient: testAPIClient(t, server),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "1 pull request approval(s) failed") {
		t.Fatalf("runApproval() error = %v, want aggregate approval error", err)
	}
	wantSummary := ApprovalSummary{Listed: 5, Approved: 1, Skipped: 3, Failed: 1}
	if summary != wantSummary {
		t.Errorf("summary = %+v, want %+v", summary, wantSummary)
	}
	if !reflect.DeepEqual(postCounts, map[int]int{3: 1, 4: 1, 5: 1}) {
		t.Errorf("POST counts = %#v", postCounts)
	}
	for _, text := range []string{
		"scanning all 1 accessible repositories",
		"Summary: listed=5 approved=1 skipped=3 failed=1",
	} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("output %q does not contain %q", output.String(), text)
		}
	}
}
