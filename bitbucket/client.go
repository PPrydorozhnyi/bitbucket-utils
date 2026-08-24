package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 30 * time.Second
	// todo stream json with json.Decoder
	// 32 MiB per successful page
	maxResponseBody = 32 << 20
	// 64 KiB per error response
	maxErrorBody = 64 << 10
)

type client struct {
	httpClient  *http.Client
	credentials Credentials
}

type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("%s %s: Bitbucket API returned %s", e.Method, e.URL, e.Status)
	}
	return fmt.Sprintf("%s %s: Bitbucket API returned %s: %s", e.Method, e.URL, e.Status, e.Message)
}

type apiPage[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}

func newClient(httpClient *http.Client, credentials Credentials) *client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &client{
		httpClient:  httpClient,
		credentials: credentials,
	}
}

func (c *client) doJSON(
	ctx context.Context,
	method string,
	rawURL string,
	result any,
) error {
	request, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create Bitbucket request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(c.credentials.User, c.credentials.Token)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: execute Bitbucket request: %w", method, rawURL, err)
	}

	limit := int64(maxResponseBody)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		limit = maxErrorBody
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return fmt.Errorf("%s %s: read Bitbucket response: %w", method, rawURL, readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%s %s: close Bitbucket response: %w", method, rawURL, closeErr)
	}
	if int64(len(body)) > limit {
		return fmt.Errorf("%s %s: Bitbucket response exceeds %d bytes", method, rawURL, limit)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(method, rawURL, response, body)
	}
	if result == nil {
		return nil
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return fmt.Errorf("%s %s: decode Bitbucket response: empty response body", method, rawURL)
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("%s %s: decode Bitbucket response: %w", method, rawURL, err)
	}
	return nil
}

func getAllPages[T any](
	ctx context.Context,
	client *client,
	firstURL string,
) ([]T, error) {
	var all []T
	next := firstURL
	visited := make(map[string]struct{})

	for next != "" {
		if _, seen := visited[next]; seen {
			return nil, fmt.Errorf("paginate Bitbucket response: next-page cycle at %s", next)
		}
		visited[next] = struct{}{}

		var page apiPage[T]
		if err := client.doJSON(ctx, http.MethodGet, next, &page); err != nil {
			return nil, fmt.Errorf("fetch Bitbucket page: %w", err)
		}
		all = append(all, page.Values...)

		resolved, err := resolveNextPage(next, page.Next)
		if err != nil {
			return nil, err
		}
		next = resolved
	}
	return all, nil
}

func resolveNextPage(current, next string) (string, error) {
	if next == "" {
		return "", nil
	}

	currentURL, err := url.Parse(current)
	if err != nil {
		return "", fmt.Errorf("parse current Bitbucket page URL: %w", err)
	}
	nextURL, err := currentURL.Parse(next)
	if err != nil {
		return "", fmt.Errorf("parse next Bitbucket page URL: %w", err)
	}
	if !strings.EqualFold(currentURL.Scheme, nextURL.Scheme) ||
		!strings.EqualFold(currentURL.Host, nextURL.Host) {
		return "", fmt.Errorf(
			"paginate Bitbucket response: next page changed origin from %s://%s to %s://%s",
			currentURL.Scheme,
			currentURL.Host,
			nextURL.Scheme,
			nextURL.Host,
		)
	}
	return nextURL.String(), nil
}

func newAPIError(method, rawURL string, response *http.Response, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
			Detail  string `json:"detail"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)

	message := strings.TrimSpace(envelope.Error.Message)
	detail := strings.TrimSpace(envelope.Error.Detail)
	if message == "" {
		message = detail
	} else if detail != "" && detail != message {
		message += ": " + detail
	}
	if message == "" {
		message = strings.TrimSpace(string(body))
	}

	return &APIError{
		Method:     method,
		URL:        rawURL,
		StatusCode: response.StatusCode,
		Status:     response.Status,
		Message:    message,
	}
}

func isAPIStatus(err error, statusCode int) bool {
	apiError, ok := errors.AsType[*APIError](err)
	return ok && apiError.StatusCode == statusCode
}
