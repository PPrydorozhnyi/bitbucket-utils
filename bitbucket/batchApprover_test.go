package bitbucket

import (
	"strings"
	"testing"
)

func TestApproveAcceptsNoAuthorURLBeforePendingRepositoryScan(t *testing.T) {
	err := Approve(
		"https://bitbucket.org/team/workspace/pull-requests/?project=%7Bproject-uuid%7D&state=OPEN",
	)
	if err == nil {
		t.Fatal("Approve() error = nil")
	}
	if !strings.Contains(err.Error(), "repository scan is not implemented yet") {
		t.Errorf("Approve() error = %q, want pending repository-scan error", err)
	}
}

func TestApproveAcceptsSinglePRURLBeforePendingApprovalFlow(t *testing.T) {
	err := Approve("https://bitbucket.org/team/repository/pull-requests/42")
	if err == nil {
		t.Fatal("Approve() error = nil")
	}
	if !strings.Contains(err.Error(), "single pull request approval is not implemented yet") {
		t.Errorf("Approve() error = %q, want pending single-PR error", err)
	}
}
