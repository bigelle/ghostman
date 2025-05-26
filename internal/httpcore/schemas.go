package httpcore

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func NewHttpRequest(urlArg, method string) (*HttpRequest, error) {
	req := HttpRequest{
		Method:      method,
		URL:         urlArg,
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),
		Cookies:     make(map[string]string),

		ShouldDumpRequest:   false,
		ShouldDumpResponse:  false,
		ShouldSendRequest:   true,
		ShouldSanitizeQuery: true,
	}

	host, err := extractHost(urlArg)
	if err != nil {
		return nil, err
	}
	req.Host = host

	q, err := extractQueryParams(urlArg)
	for k, v := range q {
		req.AddQueryParam(k, v...)
	}

	return &req, nil
}

func NewHttpRequestFromJSON(j []byte) (*HttpRequest, error) {
	var req HttpRequest
	r := bytes.NewReader(j)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

type HttpRequest struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Host        string              `json:"host"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Cookies     map[string]string   `json:"cookies"`
	Body        HttpBody            `json:"body"`

	// runtime opts
	ShouldDumpRequest     bool `json:"should_dump_request"`
	ShouldDumpResponse    bool `json:"should_dump_response"`
	ShouldSendRequest     bool `json:"should_send_request"`
	ShouldSanitizeQuery   bool `json:"should_sanitize_query"`
	ShouldSanitizeHeaders bool `json:"should_sanitize_headers"`
	ShouldSanitizeCookies bool `json:"should_sanitize_cookies"`
}

func (h HttpRequest) ToHTTP() (*http.Request, error) {
	req, err := http.NewRequest( // NOTE: shoud i use with context? and why?
		strings.ToUpper(h.Method),
		h.URL,
		nil,
	)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for key, val := range h.QueryParams {
		for _, v := range val {
			q.Add(key, v)
		}
	}
	req.URL.RawQuery = q.Encode()

	req.Header = h.Headers
	if h.Body.ContentType != "" && h.Body.Body != "" {
		req.Header.Add("Content-Type", h.Body.ContentType)
		// FIXME: should switch between known content types and properly set the Body
		// according to the content type
		req.Body = io.NopCloser(strings.NewReader(h.Body.Body))
	}
	for k, v := range h.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	return req, nil
}

// TODO: set, get, del, remove
func (h *HttpRequest) AddQueryParam(key string, val ...string) {
	if h.ShouldSanitizeQuery && len(val) == 0 {
		return
	}
	h.QueryParams[key] = append(h.QueryParams[key], val...)
}

// TODO: set, get, del, remove
func (h *HttpRequest) AddHeader(key string, val ...string) {
	if h.ShouldSanitizeHeaders && len(val) == 0 {
		return
	}
	h.Headers[key] = append(h.Headers[key], val...)
}

// TODO: set, get, del
func (h *HttpRequest) AddCookie(key string, val string) {
	if h.ShouldSanitizeCookies && len(val) == 0 {
		return
	}
	h.Cookies[key] = val
}

type HttpBody struct {
	ContentType string `json:"content_type"`
	Body        string `json:"body"`

	bodyR io.Reader
}

// TODO: body.Set() that accepts content type and related thing and creates proper reader
