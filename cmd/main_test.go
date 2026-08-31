package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    cliOptions
		wantErr string
	}{
		{
			name: "approve",
			args: []string{
				"--approve-pr",
				"https://bitbucket.org/team/repository/pull-requests/42",
			},
			want: cliOptions{
				Command: commandApprove,
				URL:     "https://bitbucket.org/team/repository/pull-requests/42",
			},
		},
		{
			name: "short approve dry run",
			args: []string{
				"-ap",
				"--dry-run",
				"https://bitbucket.org/team/workspace/pull-requests/",
			},
			want: cliOptions{
				Command: commandApprove,
				URL:     "https://bitbucket.org/team/workspace/pull-requests/",
				DryRun:  true,
			},
		},
		{
			name:    "missing command",
			wantErr: "no command provided",
		},
		{
			name:    "missing URL",
			args:    []string{"--approve-pr", "--dry-run"},
			wantErr: "requires a Bitbucket URL",
		},
		{
			name:    "unknown option",
			args:    []string{"--approve-pr", "--force"},
			wantErr: `unknown option "--force"`,
		},
		{
			name:    "duplicate dry run",
			args:    []string{"--approve-pr", "url", "--dry-run", "--dry-run"},
			wantErr: "--dry-run must appear at most once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCLI(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCLI() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCLI() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parseCLI() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRunHelpAndUsageError(t *testing.T) {
	t.Parallel()

	t.Run("help", func(t *testing.T) {
		t.Parallel()

		var output, errorOutput bytes.Buffer
		if code := run([]string{"--help"}, &output, &errorOutput); code != 0 {
			t.Fatalf("run() code = %d, want 0", code)
		}
		if !strings.Contains(output.String(), "BITBUCKET_API_TOKEN") {
			t.Errorf("help output = %q", output.String())
		}
		if errorOutput.Len() != 0 {
			t.Errorf("error output = %q", errorOutput.String())
		}
	})

	t.Run("usage error", func(t *testing.T) {
		t.Parallel()

		var output, errorOutput bytes.Buffer
		if code := run(nil, &output, &errorOutput); code != 2 {
			t.Fatalf("run() code = %d, want 2", code)
		}
		if !strings.Contains(errorOutput.String(), "no command provided") {
			t.Errorf("error output = %q", errorOutput.String())
		}
	})
}
