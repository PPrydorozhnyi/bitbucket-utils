# Console utilities

Go command-line utilities. Current Bitbucket command accepts full workspace pull-request dashboard URLs and single pull-request URLs.

## Bitbucket pull-request approval

```bash
export BITBUCKET_USER='you@example.com'
export BITBUCKET_TOKEN='your-api-token'

go run ./cmd --approve-pr \
  'https://bitbucket.org/delasport/workspace/pull-requests/?author=712020%3A47ee3e5f-1b5c-4616-9f46-77e67b1894cf&state=OPEN%2BDRAFT%2BQUEUED&user_filter=ALL&project=%7Ba747af2d-ade5-4e49-92c1-b6bc6480a16a%7D&query=jacoco'
```

Use an Atlassian API token. Bitbucket app passwords were disabled on June 9, 2026.

## URL parsing

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

The API query builder:

- requests 50 pull requests or 100 repositories per page
- preserves `page` on pull-request list requests
- combines classic states with `OR`
- includes the OPEN draft-visibility clause when `DRAFT` or `QUEUED` is selected
- adds project and escaped title predicates when applicable
- excludes pull requests authored by the authenticated account
- omits the project predicate from per-repository PR queries after repositories were already project-filtered

The Bitbucket HTTP client uses a 30-second default timeout, request contexts, Basic authentication, bounded response bodies, and typed API errors containing HTTP status and Bitbucket's message. List calls follow every `next` link from the requested starting page. Pagination rejects repeated URLs and cross-origin links so credentials cannot leak to another host.

Bitbucket query syntax cannot reliably express “no approved participant matching this account” across the `participants` array. The command therefore requests participant approval fields and removes both own PRs and already-approved PRs client-side before posting approvals. Repository-list and per-repository PR URL builders are ready for the no-author scan flow.

Parser accepts dashboard URLs without `author`, including project-scoped and workspace-wide forms. Repository scanning for those forms belongs to next implementation slice and currently stops before credentials or API calls. Single-PR execution is likewise not implemented yet. Existing author-based dashboard approval flow remains connected.

## Development

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```
