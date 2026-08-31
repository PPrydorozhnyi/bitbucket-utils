package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGetAllPages(t *testing.T) {
	t.Parallel()

	var requests atomic.Int64
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		user, token, ok := request.BasicAuth()
		if !ok || user != "user@example.com" || token != "secret" {
			t.Errorf("BasicAuth() = %q, %q, %v", user, token, ok)
		}
		if got := request.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept header = %q", got)
		}

		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page") {
		case "1":
			fmt.Fprintf(
				response,
				`{"values":[{"id":1}],"next":%q,"page":1}`,
				server.URL+"/pullrequests?page=2",
			)
		case "2":
			fmt.Fprint(response, `{"values":[{"id":2}],"next":"?page=3","page":2}`)
		case "3":
			fmt.Fprint(response, `{"values":[{"id":3}],"page":3}`)
		default:
			t.Errorf("unexpected page %q", request.URL.Query().Get("page"))
			http.Error(response, "unexpected page", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	apiClient := newClient(server.Client(), &Credentials{
		User:  "user@example.com",
		Token: "secret",
	})
	got, err := getAllPages[PullRequest](
		t.Context(),
		apiClient,
		server.URL+"/pullrequests?page=1",
	)
	if err != nil {
		t.Fatalf("getAllPages() error = %v", err)
	}

	want := []PullRequest{{ID: 1}, {ID: 2}, {ID: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getAllPages() = %#v, want %#v", got, want)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("request count = %d, want 3", got)
	}
}

func TestClientReturnsTypedAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusForbidden)
		fmt.Fprint(response, `{"type":"error","error":{"message":"Access denied","detail":"Missing scope"}}`)
	}))
	defer server.Close()

	apiClient := newClient(server.Client(), &Credentials{})
	err := apiClient.doJSON(t.Context(), http.MethodGet, server.URL+"/resource", new(any))
	if err == nil {
		t.Fatal("doJSON() error = nil")
	}

	apiError, ok := errors.AsType[*APIError](err)
	if !ok {
		t.Fatalf("doJSON() error type = %T, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d", apiError.StatusCode)
	}
	if apiError.Message != "Access denied: Missing scope" {
		t.Errorf("Message = %q", apiError.Message)
	}
	if !isAPIStatus(err, http.StatusForbidden) {
		t.Error("isAPIStatus() = false, want true")
	}
}

func TestClientReportsInvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(response, "{invalid")
	}))
	defer server.Close()

	apiClient := newClient(server.Client(), &Credentials{})
	err := apiClient.doJSON(t.Context(), http.MethodGet, server.URL, new(apiPage[PullRequest]))
	if err == nil {
		t.Fatal("doJSON() error = nil")
	}
	if !strings.Contains(err.Error(), "decode Bitbucket response") {
		t.Errorf("doJSON() error = %q", err)
	}
}

func TestClientAcceptsEmptySuccessfulPOST(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", request.Method)
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	apiClient := newClient(server.Client(), &Credentials{})
	if err := apiClient.doJSON(t.Context(), http.MethodPost, server.URL, nil); err != nil {
		t.Fatalf("doJSON() error = %v", err)
	}
}

func TestGetAllPagesRejectsOriginChange(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(response, `{"values":[],"next":"https://example.com/next"}`)
	}))
	defer server.Close()

	apiClient := newClient(server.Client(), &Credentials{})
	_, err := getAllPages[PullRequest](t.Context(), apiClient, server.URL)
	if err == nil {
		t.Fatal("getAllPages() error = nil")
	}
	if !strings.Contains(err.Error(), "next page changed origin") {
		t.Errorf("getAllPages() error = %q", err)
	}
}

func TestGetAllPagesRejectsCycle(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(response, `{"values":[],"next":%q}`, server.URL)
	}))
	defer server.Close()

	apiClient := newClient(server.Client(), &Credentials{})
	_, err := getAllPages[PullRequest](t.Context(), apiClient, server.URL)
	if err == nil {
		t.Fatal("getAllPages() error = nil")
	}
	if !strings.Contains(err.Error(), "next-page cycle") {
		t.Errorf("getAllPages() error = %q", err)
	}
}

func TestClientPreservesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	apiClient := newClient(&http.Client{}, &Credentials{})
	err := apiClient.doJSON(ctx, http.MethodGet, "https://example.com", new(any))
	if err == nil {
		t.Fatal("doJSON() error = nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("doJSON() error = %v, want context.Canceled", err)
	}
}
