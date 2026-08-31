package bitbucket

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func testAPIClient(t *testing.T, server *httptest.Server) *client {
	t.Helper()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	transport := server.Client().Transport
	httpClient := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			clone := request.Clone(request.Context())
			clonedURL := *request.URL
			clonedURL.Scheme = target.Scheme
			clonedURL.Host = target.Host
			clone.URL = &clonedURL
			clone.Host = ""
			return transport.RoundTrip(clone)
		}),
	}
	return newClient(httpClient, &Credentials{
		User:  "user@example.com",
		Token: "secret",
	})
}

func assertBasicAuth(t *testing.T, request *http.Request) {
	t.Helper()

	user, token, ok := request.BasicAuth()
	if !ok || user != "user@example.com" || token != "secret" {
		t.Errorf("BasicAuth() = %q, %q, %t", user, token, ok)
	}
}

func writeAPIError(response http.ResponseWriter, status int, message string) {
	response.WriteHeader(status)
	fmt.Fprintf(response, `{"error":{"message":%q}}`, message)
}

func approvalPathID(path string) (int, error) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 || segments[len(segments)-1] != "approve" {
		return 0, fmt.Errorf("unexpected approval path %q", path)
	}
	return strconv.Atoi(segments[len(segments)-2])
}
