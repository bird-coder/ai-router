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
}

type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
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

func (req *HTTPRequest) SetBodyBytes(body []byte) {
	req.Body = bytes.NewReader(body)
}

func (req *HTTPRequest) SetBodyString(body string) {
	req.Body = strings.NewReader(body)
}

func (req *HTTPRequest) SetJSONBody(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal request body failed for %T: %w", value, err)
	}
	req.SetBodyBytes(payload)
	req.withDefaultContentType(contentTypeJson)
	return nil
}

func (req *HTTPRequest) SetFormBody(form url.Values) {
	req.SetBodyString(form.Encode())
	req.withDefaultContentType(contentTypeForm)
}

func (req HTTPRequest) Do(ctx context.Context) (*HTTPResponse, error) {
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
		return nil, fmt.Errorf("%s %s failed: %w", req.Method, targetURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s %s response failed: %w", req.Method, targetURL, err)
	}

	if !req.statusAllowed(resp.StatusCode) {
		return nil, fmt.Errorf("%s %s returned status %d: %s", req.Method, targetURL, resp.StatusCode, string(respBody))
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

func (req *HTTPRequest) withDefaultContentType(contentType string) {
	if _, ok := req.Headers[contentTypeKey]; ok {
		return
	}
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers[contentTypeKey] = contentType
}

func HttpHead(targetURL string, headers map[string]string) error {
	req := HTTPRequest{
		Method:         http.MethodHead,
		URL:            targetURL,
		Headers:        headers,
		ExpectedStatus: []int{http.StatusOK, http.StatusNoContent, http.StatusMovedPermanently, http.StatusFound},
	}
	req.withDefaultContentType(contentTypeJson)

	_, err := req.Do(context.Background())
	return err
}

func HttpGet(targetURL string, query map[string]any, headers map[string]string) ([]byte, error) {
	req := HTTPRequest{
		Method:         http.MethodGet,
		URL:            targetURL,
		Query:          query,
		Headers:        headers,
		ExpectedStatus: []int{http.StatusOK},
	}
	req.withDefaultContentType(contentTypeJson)

	resp, err := req.Do(context.Background())
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func HttpPostJson(targetURL string, data any, headers map[string]string) ([]byte, error) {
	req := HTTPRequest{
		Method:         http.MethodPost,
		URL:            targetURL,
		Headers:        headers,
		ExpectedStatus: []int{http.StatusOK},
	}
	if err := req.SetJSONBody(data); err != nil {
		return nil, err
	}

	resp, err := req.Do(context.Background())
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

	req := HTTPRequest{
		Method:         http.MethodPost,
		URL:            targetURL,
		Headers:        headers,
		ExpectedStatus: []int{http.StatusOK},
	}
	req.SetFormBody(form)

	resp, err := req.Do(context.Background())
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
