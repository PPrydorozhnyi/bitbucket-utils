package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const wUrl = "https://api.bitbucket.org/2.0/workspaces"

func Approve(u string) error {
	return approve(context.Background(), u)
}

func approve(ctx context.Context, u string) error {
	fmt.Printf("Approving pull request for %s\n", u)

	up, err := parseURL(u)
	if err != nil {
		return err
	}
	for _, key := range up.UnknownParams {
		fmt.Printf("Warning: unsupported query parameter %q was not applied\n", key)
	}
	if up.Mode != URLModeDashboard {
		return fmt.Errorf("single pull request approval is not implemented yet")
	}
	if up.Filters.Author == "" {
		return fmt.Errorf("dashboard URL without author parsed successfully; repository scan is not implemented yet")
	}

	prPath, err := buildPrUrl(up)
	if err != nil {
		return err
	}

	creds, err := getCreds()
	if err != nil {
		return err
	}
	apiClient := newClient(nil, *creds)

	al, err := getPRsToApprove(ctx, apiClient, prPath)
	if err != nil {
		return err
	}

	err = approveAll(ctx, apiClient, al)
	if err != nil {
		return err
	}

	return nil
}

func buildPrUrl(up *URLParams) (string, error) {
	u, err := url.Parse(wUrl)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(up.Workspace, "pullrequests", up.Filters.Author)
	u.RawQuery = buildPullRequestQuery(up.Filters, true).Encode()
	return u.String(), nil
}
func getCreds() (*Credentials, error) {
	user := os.Getenv("BITBUCKET_USER")

	if user == "" {
		return nil, fmt.Errorf("BITBUCKET_USER environment variable is not set")
	}

	token := os.Getenv("BITBUCKET_TOKEN")

	if token == "" {
		return nil, fmt.Errorf("BITBUCKET_TOKEN environment variable is not set")
	}

	return &Credentials{User: user, Token: token}, nil
}

func getPRsToApprove(
	ctx context.Context,
	apiClient *client,
	url string,
) ([]PullRequest, error) {
	return getAllPages[PullRequest](ctx, apiClient, url)
}

func approveAll(ctx context.Context, apiClient *client, prs []PullRequest) error {
	for _, pr := range prs {
		if err := apiClient.doJSON(ctx, http.MethodPost, pr.Links.Approve.Href, nil); err != nil {
			return err
		}
		fmt.Printf("Approved pull request: %s. Check it out: %s\n", pr.Title, pr.Links.Html.Href)
	}

	return nil
}
