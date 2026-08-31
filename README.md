# Console utilities

Go command-line utilities. The Bitbucket command accepts full workspace pull-request dashboard URLs and single pull-request URLs.

## Bitbucket pull-request approval

Create an Atlassian API token with scopes in Atlassian account settings. Required scopes:

- `read:user:bitbucket`
- `read:repository:bitbucket`
- `read:pullrequest:bitbucket`
- `write:pullrequest:bitbucket`

Bitbucket app passwords were disabled on June 9, 2026.

Export the email address and API token used for Basic authentication:

```bash
export BITBUCKET_EMAIL='you@example.com'
export BITBUCKET_API_TOKEN='your-api-token'
```

## Run a prebuilt binary

Download the matching executable from the repository's [`build`](build/) directory:

- Apple Silicon macOS: `console-utils-darwin-arm64`
- Intel macOS: `console-utils-darwin-amd64`
- 64-bit Windows: `console-utils-windows-amd64.exe`

On macOS, make the downloaded binary executable and run it:

```bash
chmod +x console-utils-darwin-arm64

./console-utils-darwin-arm64 --approve-pr \
  'https://bitbucket.org/sport/repository/pull-requests/42' \
  --dry-run
```

Use `console-utils-darwin-amd64` instead on an Intel Mac.

On Windows PowerShell:

```powershell
$env:BITBUCKET_EMAIL = 'you@example.com'
$env:BITBUCKET_API_TOKEN = 'your-api-token'

.\console-utils-windows-amd64.exe --approve-pr `
  'https://bitbucket.org/sport/repository/pull-requests/42' `
  --dry-run
```

Remove `--dry-run` only after reviewing the listed pull requests. macOS binaries are not signed or notarized; only run artifacts obtained from a repository you trust.

## Run from source

Preview a dashboard batch without approving anything:

```bash
go run ./cmd --approve-pr \
  'https://bitbucket.org/sport/workspace/pull-requests/?author=712020%3A47ee3e5f-1b5c-4616-9f46-77e67b1894cf&state=OPEN%2BDRAFT%2BQUEUED&user_filter=ALL&project=%7Ba747af2d-ade5-4e49-92c1-b6bc6480a16a%7D&query=jacoco' \
  --dry-run
```

Approve the resulting dashboard batch by removing `--dry-run`:

```bash
go run ./cmd --approve-pr \
  'https://bitbucket.org/sport/workspace/pull-requests/?author=712020%3A47ee3e5f-1b5c-4616-9f46-77e67b1894cf&state=OPEN%2BDRAFT%2BQUEUED&user_filter=ALL&project=%7Ba747af2d-ade5-4e49-92c1-b6bc6480a16a%7D&query=jacoco'
```

Single pull requests use the same command:

```bash
go run ./cmd --approve-pr \
  'https://bitbucket.org/sport/repository/pull-requests/42' \
  --dry-run
```

The command prints parsed filters first and finishes with:

```text
Summary: listed=4 approved=2 skipped=1 failed=1
```

Approval failures do not stop the remaining batch. The process exits nonzero after the batch if any approval failed. Authentication, listing, and malformed-response failures stop immediately.

## URLs and filtering

Supported forms:

- Dashboard: `https://bitbucket.org/{workspace}/workspace/pull-requests/`
- Single PR: `https://bitbucket.org/{workspace}/{repository}/pull-requests/{id}`

Recognized dashboard parameters:

- `author`
- `state`: `OPEN`, `DRAFT`, `QUEUED`, `MERGED`, `DECLINED`, or `SUPERSEDED`
- `project`
- `query`
- `user_filter`: `ALL`, `AUTHOR`, `REVIEWING`, or `PARTICIPATING`
- `reviewer`
- `page`: positive API page number; omitted means first page

`state` accepts encoded separators such as `OPEN%2BDRAFT`, literal separators such as `OPEN+DRAFT`, and repeated parameters. Missing state defaults to `OPEN`; missing `user_filter` defaults to `REVIEWING`. Unknown parameters produce warnings.

Filtering behavior:

- `OPEN` matches open pull requests that are neither draft nor queued.
- `DRAFT` and `QUEUED` match their corresponding open flags.
- `MERGED`, `DECLINED`, and `SUPERSEDED` match the exact API state.
- `project` matches the destination repository project UUID.
- `query` is escaped in the API title query and reapplied case-insensitively to returned titles and descriptions.
- `reviewer` matches a reviewer account ID or UUID.
- `user_filter=ALL` applies no current-user role filter.
- `user_filter=REVIEWING` requires the current user in `reviewers`.
- `user_filter=PARTICIPATING` requires the current user in `participants`.
- `user_filter=AUTHOR` without an explicit `author` produces no candidates because the current user cannot approve their own pull requests.

The command completes all listing and filtering before any approval request. It skips pull requests authored by the authenticated user, pull requests already approved by that user, and conflicts reported as already approved.

## Dashboard listing strategies

- With `author`, the command uses Bitbucket's workspace pull-requests-by-author endpoint. Account IDs containing `:` retry with their UUID suffix after a 404.
- Without `author` but with `project`, the command lists project repositories and then lists pull requests in each repository.
- Without `author` or `project`, the command scans every accessible repository in the workspace and prints a warning with the repository count. This can be slow and consume many API requests.

Lists use 50 pull requests or 100 repositories per page and follow all `next` links. The optional dashboard `page` is preserved on pull-request list requests. Pagination rejects cycles and cross-origin links. If Bitbucket rejects an author query's nested project predicate, the command retries without that predicate and enforces project matching client-side.

The HTTP client uses a 30-second timeout, request contexts, bounded response bodies, Basic authentication, and typed API errors containing Bitbucket's status and message.

## Development

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```
