package util

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGetBuildsQueryAndReturnsBody(t *testing.T) {
	restore := swapHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if got := r.URL.Query().Get("q"); got != "router" {
				t.Fatalf("q = %q, want %q", got, "router")
			}
			if got := r.Header.Get("X-Test"); got != "yes" {
				t.Fatalf("header = %q, want %q", got, "yes")
			}
			return responseOf(http.StatusOK, `{"ok":true}`), nil
		}),
	})
	defer restore()

	body, err := HttpGet("https://example.com/search", map[string]any{"q": "router"}, map[string]string{"X-Test": "yes"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s", string(body))
	}
}

func TestPostJSONSendsJSONBody(t *testing.T) {
	restore := swapHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
				t.Fatalf("content-type = %q", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload["model"] != "qwen-plus" {
				t.Fatalf("model = %v", payload["model"])
			}
			return responseOf(http.StatusOK, `{"choices":[{"text":"ok"}]}`), nil
		}),
	})
	defer restore()

	body, err := HttpPostJson("https://example.com/chat", map[string]any{"model": "qwen-plus"}, nil)
	if err != nil {
		t.Fatalf("PostJSON() error = %v", err)
	}
	if string(body) != `{"choices":[{"text":"ok"}]}` {
		t.Fatalf("body = %s", string(body))
	}
}

func TestDoSendsRawBody(t *testing.T) {
	restore := swapHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if string(data) != `{"raw":true}` {
				t.Fatalf("body = %s, want raw json", string(data))
			}
			return responseOf(http.StatusOK, `{"ok":true}`), nil
		}),
	})
	defer restore()

	req := HTTPRequest{
		Method:         http.MethodPost,
		URL:            "https://example.com/raw",
		Body:           strings.NewReader(`{"raw":true}`),
		Headers:        map[string]string{"Content-Type": "application/json"},
		ExpectedStatus: []int{http.StatusOK},
	}
	body, err := req.Do(context.Background())
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if string(body.Body) != `{"ok":true}` {
		t.Fatalf("body = %s", string(body.Body))
	}
}

func TestDoReturnsErrorOnUnexpectedStatus(t *testing.T) {
	restore := swapHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return responseOf(http.StatusBadGateway, "bad gateway"), nil
		}),
	})
	defer restore()

	_, err := HttpGet("https://example.com/fail", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("error = %v, want status code", err)
	}
}

func TestSetBodyHelpers(t *testing.T) {
	req := &HTTPRequest{}
	req.SetBodyString("hello")

	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read string body: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("string body = %q", string(data))
	}

	req.SetBodyBytes([]byte("world"))
	data, err = io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read bytes body: %v", err)
	}
	if string(data) != "world" {
		t.Fatalf("bytes body = %q", string(data))
	}
}

func TestSetJSONBody(t *testing.T) {
	req := &HTTPRequest{}
	if err := req.SetJSONBody(map[string]any{"model": "qwen-plus"}); err != nil {
		t.Fatalf("SetJSONBody() error = %v", err)
	}
	if got := req.Headers[contentTypeKey]; got != contentTypeJson {
		t.Fatalf("content-type = %q, want %q", got, contentTypeJson)
	}

	var payload map[string]any
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode json body: %v", err)
	}
	if payload["model"] != "qwen-plus" {
		t.Fatalf("model = %v", payload["model"])
	}
}

func TestSetFormBody(t *testing.T) {
	req := &HTTPRequest{}
	form := url.Values{
		"name":  []string{"router"},
		"level": []string{"1"},
	}
	req.SetFormBody(form)

	if got := req.Headers[contentTypeKey]; got != contentTypeForm {
		t.Fatalf("content-type = %q, want %q", got, contentTypeForm)
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read form body: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "name=router") || !strings.Contains(body, "level=1") {
		t.Fatalf("form body = %q", body)
	}
}

func TestCloneCopiesMutableFields(t *testing.T) {
	original := HTTPRequest{
		Method:         http.MethodPost,
		URL:            "https://example.com",
		Query:          map[string]any{"page": 1},
		Headers:        map[string]string{"X-Test": "yes"},
		ExpectedStatus: []int{http.StatusOK},
		Body:           strings.NewReader("body"),
	}

	cloned := original.Clone()
	cloned.Query["page"] = 2
	cloned.Headers["X-Test"] = "no"
	cloned.ExpectedStatus[0] = http.StatusCreated

	if original.Query["page"] != 1 {
		t.Fatalf("original query mutated: %v", original.Query["page"])
	}
	if original.Headers["X-Test"] != "yes" {
		t.Fatalf("original headers mutated: %v", original.Headers["X-Test"])
	}
	if original.ExpectedStatus[0] != http.StatusOK {
		t.Fatalf("original expected status mutated: %v", original.ExpectedStatus[0])
	}
	if cloned.Body != original.Body {
		t.Fatal("body reader should be shared in clone")
	}
}

func TestStringifyScalars(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil", input: nil, want: ""},
		{name: "string", input: "abc", want: "abc"},
		{name: "bool", input: true, want: "true"},
		{name: "int", input: 12, want: "12"},
		{name: "uint64", input: uint64(34), want: "34"},
		{name: "float32", input: float32(1.25), want: strconv.FormatFloat(1.25, 'f', -1, 32)},
		{name: "float64", input: 2.5, want: strconv.FormatFloat(2.5, 'f', -1, 64)},
	}

	for _, tt := range tests {
		if got := stringify(tt.input); got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func swapHTTPClient(mock *http.Client) func() {
	original := client
	client = mock
	return func() {
		client = original
	}
}

func responseOf(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
