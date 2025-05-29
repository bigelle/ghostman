package httpcore

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	q, err := ExtractQueryParams(urlArg)
	if err != nil {
		return nil, err
	}
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
	// serializable
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Cookies     map[string]string   `json:"cookies"`

	// runtime opts
	ShouldDumpRequest     bool `json:"should_dump_request"`
	ShouldDumpResponse    bool `json:"should_dump_response"`
	ShouldSendRequest     bool `json:"should_send_request"`
	ShouldSanitizeQuery   bool `json:"should_sanitize_query"`
	ShouldSanitizeHeaders bool `json:"should_sanitize_headers"`
	ShouldSanitizeCookies bool `json:"should_sanitize_cookies"`

	// only through flags or methods
	body HttpBody
}

func (h HttpRequest) ToHTTP() (*http.Request, error) {
	var body io.Reader
	if !h.body.IsEmpty() {
		if err := h.body.Setup(); err != nil {
			return nil, err
		}
		body = bytes.NewReader(h.body.Source)
	}

	req, err := http.NewRequest( // NOTE: shoud i use with context? and why?
		strings.ToUpper(h.Method),
		h.URL,
		body,
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

	for k, vals := range h.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	for k, v := range h.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	if !h.body.IsEmpty() {
		if err := h.body.Setup(); err != nil {
			return nil, err
		}
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

func (h *HttpRequest) SetBodyJSON(b []byte) error {
	if !IsValidJSON(b) {
		return fmt.Errorf("not a valid JSON")
	}
	h.body = HttpBody{
		ContentType: "application/json",
		Source:      b,
	}
	return nil
}

type HttpBody struct {
	ContentType string
	Source      []byte

	bodyR io.Reader
}

func (h HttpBody) IsEmpty() bool {
	return h.ContentType == "" && len(h.Source) == 0
}

// TODO: body.Setup() that creates proper reader
func (h *HttpBody) Setup() error {
	switch h.ContentType {
	case "application/json":
		return h.setupJSON()
	default:
		return fmt.Errorf("unknown content type: %s", h.ContentType)
	}
}

func (h *HttpBody) setupJSON() error {
	if !IsValidJSON(h.Source) {
		return fmt.Errorf("not a valid JSON")
	}
	buf := bytes.NewReader(h.Source)
	h.bodyR = buf
	return nil
}

func IsValidJSON(buf []byte) bool {
	trimmed := bytes.TrimSpace(buf)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if json.Valid(trimmed) {
			return true
		}
	}
	return false
}
