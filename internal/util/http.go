package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	defaultTimeout = 30 * time.Second

	client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          500,
			MaxConnsPerHost:       200,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: defaultTimeout,
	}

	contentTypeKey  = "Content-Type"
	contentTypeJson = "application/json"
	contentTypeForm = "application/x-www-form-urlencoded"
)

type HTTPRequest struct {
	Method         string
	URL            string
	Query          map[string]any
	Headers        map[string]string
	Body           io.Reader
	ExpectedStatus []int
	rawBodyString  string

	err error
}

type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func NewHttpRequest(url string, method string) *HTTPRequest {
	req := &HTTPRequest{
		Method: method,
		URL:    url,
	}
	return req
}

func (req HTTPRequest) Clone() HTTPRequest {
	cloned := req
	if req.Query != nil {
		cloned.Query = make(map[string]any, len(req.Query))
		for key, value := range req.Query {
			cloned.Query[key] = value
		}
	}
	if req.Headers != nil {
		cloned.Headers = make(map[string]string, len(req.Headers))
		for key, value := range req.Headers {
			cloned.Headers[key] = value
		}
	}
	if req.ExpectedStatus != nil {
		cloned.ExpectedStatus = append([]int(nil), req.ExpectedStatus...)
	}
	return cloned
}

func (req *HTTPRequest) WithMethod(method string) *HTTPRequest {
	req.Method = method
	return req
}

func (req *HTTPRequest) WithURL(rawURL string) *HTTPRequest {
	req.URL = rawURL
	return req
}

func (req *HTTPRequest) WithHeaders(headers map[string]string) *HTTPRequest {
	if headers == nil {
		req.Headers = nil
		return req
	}
	req.Headers = make(map[string]string, len(headers))
	for key, value := range headers {
		req.Headers[key] = value
	}
	return req
}

func (req *HTTPRequest) WithHeader(key, value string) *HTTPRequest {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers[key] = value
	return req
}

func (req *HTTPRequest) WithQuery(query map[string]any) *HTTPRequest {
	if query == nil {
		req.Query = nil
		return req
	}
	req.Query = make(map[string]any, len(query))
	for key, value := range query {
		req.Query[key] = value
	}
	return req
}

func (req *HTTPRequest) WithExpectedStatus(status ...int) *HTTPRequest {
	req.ExpectedStatus = append([]int(nil), status...)
	return req
}

func (req *HTTPRequest) WithBodyBytes(body []byte) *HTTPRequest {
	req.setBodyBytes(body)
	return req
}

func (req *HTTPRequest) WithBodyString(body string) *HTTPRequest {
	req.setBodyString(body)
	return req
}

func (req *HTTPRequest) WithJSONBody(value any) *HTTPRequest {
	if req.err != nil {
		return req
	}
	payload, err := json.Marshal(value)
	if err != nil {
		req.err = fmt.Errorf("marshal request body failed for %T: %w", value, err)
		return req
	}
	req.setBodyBytes(payload)
	req.setDefaultContentType(contentTypeJson)
	return req
}

func (req *HTTPRequest) WithFormBody(form url.Values) *HTTPRequest {
	req.setBodyString(form.Encode())
	req.setDefaultContentType(contentTypeForm)
	return req
}

func (req HTTPRequest) Do(ctx context.Context) (*HTTPResponse, error) {
	if req.err != nil {
		return nil, req.err
	}

	targetURL, err := req.buildURL()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, req.Body)
	if err != nil {
		return nil, err
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http %s failed, url: %s, body: %s, error: %w", req.Method, targetURL, req.rawBodyString, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read http response failed, url: %s, body: %s, error: %w", targetURL, req.rawBodyString, err)
	}

	if !req.statusAllowed(resp.StatusCode) {
		return nil, fmt.Errorf("http %s returned status %d, url: %s, resp: %s", req.Method, resp.StatusCode, targetURL, string(respBody))
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       respBody,
	}, nil
}

func (req HTTPRequest) buildURL() (string, error) {
	if len(req.Query) == 0 {
		return req.URL, nil
	}

	parsed, err := url.Parse(req.URL)
	if err != nil {
		return "", fmt.Errorf("parse url %q failed: %w", req.URL, err)
	}

	values := parsed.Query()
	for key, value := range req.Query {
		values.Set(key, stringify(value))
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func (req HTTPRequest) statusAllowed(status int) bool {
	if len(req.ExpectedStatus) == 0 {
		return status >= 200 && status < 300
	}
	for _, candidate := range req.ExpectedStatus {
		if status == candidate {
			return true
		}
	}
	return false
}

func (req *HTTPRequest) setBodyBytes(body []byte) {
	req.rawBodyString = string(body)
	req.Body = bytes.NewReader(body)
}

func (req *HTTPRequest) setBodyString(body string) {
	req.rawBodyString = body
	req.Body = strings.NewReader(body)
}

func (req *HTTPRequest) setDefaultContentType(contentType string) {
	if _, ok := req.Headers[contentTypeKey]; ok {
		return
	}
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers[contentTypeKey] = contentType
}

func HttpHead(targetURL string, headers map[string]string) error {
	_, err := new(HTTPRequest).
		WithURL(targetURL).
		WithMethod(http.MethodHead).
		WithHeader(contentTypeKey, contentTypeJson).
		WithHeaders(headers).
		WithExpectedStatus(http.StatusOK, http.StatusNoContent, http.StatusMovedPermanently, http.StatusFound).
		Do(context.Background())

	return err
}

func HttpGet(targetURL string, query map[string]any, headers map[string]string) ([]byte, error) {
	resp, err := new(HTTPRequest).
		WithURL(targetURL).
		WithMethod(http.MethodGet).
		WithQuery(query).
		WithHeader(contentTypeKey, contentTypeJson).
		WithHeaders(headers).
		WithExpectedStatus(http.StatusOK).
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func HttpPostJSON(targetURL string, data any, headers map[string]string) ([]byte, error) {
	resp, err := new(HTTPRequest).
		WithURL(targetURL).
		WithMethod(http.MethodPost).
		WithJSONBody(data).
		WithHeaders(headers).
		WithExpectedStatus(http.StatusOK).
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func HttpPostForm(targetURL string, data map[string]any, headers map[string]string) ([]byte, error) {
	form := url.Values{}
	for key, value := range data {
		form.Set(key, stringify(value))
	}

	resp, err := new(HTTPRequest).
		WithURL(targetURL).
		WithMethod(http.MethodPost).
		WithFormBody(form).
		WithHeaders(headers).
		WithExpectedStatus(http.StatusOK).
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func BuildParams(params map[string]any) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, stringify(value))
	}
	return values.Encode()
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case float32: //浮点数需要先转成字符串，防止精度丢失
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case float64: //浮点数需要先转成字符串，防止精度丢失
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}
