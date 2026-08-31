package bitbucket

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestRunApprovalAuthorFilterWithoutAuthorShortCircuits(t *testing.T) {
	t.Parallel()

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(
		response http.ResponseWriter,
		_ *http.Request,
	) {
		requests++
		http.Error(response, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	var output bytes.Buffer
	summary, err := runApproval(
		t.Context(),
		"https://bitbucket.org/team/workspace/pull-requests/?user_filter=AUTHOR",
		approvalRunOptions{
			Output:    &output,
			APIClient: testAPIClient(t, server),
		},
	)
	if err != nil {
		t.Fatalf("runApproval() error = %v", err)
	}
	if summary != (ApprovalSummary{}) {
		t.Errorf("summary = %+v, want zero", summary)
	}
	if !strings.Contains(output.String(), "No approvable pull requests") {
		t.Errorf("output = %q", output.String())
	}
	if requests != 0 {
		t.Errorf("request count = %d, want 0", requests)
	}
}

func TestGetCredsUsesAPITokenEnvironment(t *testing.T) {
	t.Setenv("BITBUCKET_EMAIL", " user@example.com ")
	t.Setenv("BITBUCKET_API_TOKEN", "secret")

	credentials, err := getCreds()
	if err != nil {
		t.Fatalf("getCreds() error = %v", err)
	}
	want := &Credentials{User: "user@example.com", Token: "secret"}
	if !reflect.DeepEqual(credentials, want) {
		t.Errorf("getCreds() = %#v, want %#v", credentials, want)
	}
}

func TestGetCredsValidatesEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		token   string
		wantErr string
	}{
		{
			name:    "missing email",
			token:   "secret",
			wantErr: "BITBUCKET_EMAIL",
		},
		{
			name:    "missing token",
			email:   "user@example.com",
			wantErr: "BITBUCKET_API_TOKEN",
		},
		{
			name:    "blank token",
			email:   "user@example.com",
			token:   "  ",
			wantErr: "BITBUCKET_API_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BITBUCKET_EMAIL", tt.email)
			t.Setenv("BITBUCKET_API_TOKEN", tt.token)

			_, err := getCreds()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("getCreds() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
