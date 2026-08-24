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

`state` accepts encoded separators such as `OPEN%2BDRAFT`, literal separators such as `OPEN+DRAFT`, and repeated parameters. Missing state defaults to `OPEN`; missing `user_filter` defaults to `ALL`. Unknown parameters produce warnings.

The API query builder:

- requests 50 pull requests or 100 repositories per page
- preserves `page` on pull-request list requests
- combines classic states with `OR`
- includes the OPEN draft-visibility clause when `DRAFT` or `QUEUED` is selected
- adds project and escaped title predicates when applicable
- omits the project predicate from per-repository PR queries after repositories were already project-filtered

Parser accepts dashboard URLs without `author`, including project-scoped and workspace-wide forms. Repository scanning for those forms belongs to next implementation slice and currently stops before credentials or API calls. Single-PR execution is likewise not implemented yet. Existing author-based dashboard approval flow remains connected.

## Development

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```
