package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const wUrl = "https://api.bitbucket.org/2.0/workspaces"

func Approve(u string) error {
	fmt.Printf("Approving pull request for %s\n", u)

	up, err := parseUrl(u)
	if err != nil {
		return err
	}

	prPath, err := buildPrUrl(up)
	if err != nil {
		return err
	}

	creds, err := getCreds()
	if err != nil {
		return err
	}

	al, err := getPrsToApprove(prPath, creds)
	if err != nil {
		return err
	}

	err = approveAll(al, creds)
	if err != nil {
		return err
	}

	return nil
}

func buildPrUrl(up *UrlParams) (string, error) {
	u, err := url.Parse(wUrl)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(up.WorkSpace, "pullrequests", up.Author)

	q := u.Query()

	f := `state="OPEN"`

	if up.Title != "" {
		f = fmt.Sprintf(`title~"%s" AND %s`, up.Title, f)
	}

	q.Set("q", f)

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseUrl(u string) (*UrlParams, error) {

	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	q := parsed.Query()

	a := q.Get("author")

	if a == "" {
		return nil, fmt.Errorf("no author query parameter found")
	}

	title := q.Get("query")

	parsedPath := strings.Split(strings.Trim(parsed.Path, "/"), "/")

	if len(parsedPath) < 2 {
		return nil, fmt.Errorf("incorrect path %s", parsed.Path)
	}

	return &UrlParams{Author: a, WorkSpace: parsedPath[0], Title: title}, nil
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

func getPrsToApprove(url string, creds *Credentials) ([]PullRequest, error) {
	newData, err := executeRequest(url, http.MethodGet, creds)
	if err != nil {
		return nil, err
	}

	var parsed PrContainer
	err = json.Unmarshal(newData, &parsed)
	if err != nil {
		return nil, err
	}

	return parsed.Prs, nil
}

func executeRequest(url string, method string, creds *Credentials) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(creds.User, creds.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		e := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body for %s %v", url, e)
		}
	}(resp.Body)

	newData, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d for %s. response is %s", resp.StatusCode, url, string(newData))
	}

	return newData, nil
}

func approveAll(prs []PullRequest, creds *Credentials) error {

	for _, pr := range prs {
		_, err := executeRequest(pr.Links.Approve.Href, http.MethodPost, creds)
		if err != nil {
			return err
		}
		fmt.Printf("Approved pull request: %s. Check it out: %s\n", pr.Title, pr.Links.Html.Href)
	}

	return nil
}
