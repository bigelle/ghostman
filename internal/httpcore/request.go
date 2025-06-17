package httpcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bigelle/ghostman/internal/shared"
)

func NewRequest(reqURL string) (*Request, error) {
	_, err := url.ParseRequestURI(reqURL)
	if err != nil {
		return nil, err
	}
	return newRequest(reqURL, "GET")
}

func newRequest(url, method string) (*Request, error) {
	req := Request{
		Method:      method,
		URL:         url,
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),
		Options: Options{
			Verbose:         func(b bool) *bool { return &b }(false),
			SendRequest:     func(b bool) *bool { return &b }(true),
			SanitizeQuery:   func(b bool) *bool { return &b }(true),
			SanitizeHeaders: func(b bool) *bool { return &b }(true),
			SanitizeCookies: func(b bool) *bool { return &b }(true),
			Timeout:         30,
		},
	}

	q, err := ExtractQueryParams(url)
	if err != nil {
		return nil, err
	}
	for k, v := range q {
		req.AddQueryParam(k, v...)
	}

	return &req, nil
}

func NewRequestFromJSON(j []byte) (*Request, error) {
	req := Request{
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),
		Options: Options{
			Verbose:         func(b bool) *bool { return &b }(false),
			SendRequest:     func(b bool) *bool { return &b }(true),
			SanitizeQuery:   func(b bool) *bool { return &b }(true),
			SanitizeHeaders: func(b bool) *bool { return &b }(true),
			SanitizeCookies: func(b bool) *bool { return &b }(true),
			Timeout:         30,
		},
	}

	r := bytes.NewReader(j)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}

	if req.Body != nil {
		body, err := req.Body.Parse()
		if err != nil {
			return nil, err
		}
		req.SetBody(body)
	}
	return &req, nil
}

type Request struct {
	// serializable
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
	Cookies     []Cookie            `json:"cookies,omitempty"`
	Body        *BodySpec           `json:"body,omitempty"`

	// runtime opts
	Options Options `json:"options"`

	// only through flags or methods
	body Body
}

type Options struct {
	Verbose         *bool `json:"verbose,omitempty"`
	SendRequest     *bool `json:"send_request,omitempty"`
	SanitizeQuery   *bool `json:"sanitize_query,omitempty"`
	SanitizeHeaders *bool `json:"sanitize_headers,omitempty"`
	SanitizeCookies *bool `json:"sanitize_cookies,omitempty"`
	Timeout         int   `json:"timeout,omitempty"`
}

func (r Request) String() string {
	return "" // FIXME:
}

func (h Request) IsEmptyBody() bool {
	return h.body == nil
}

func (h Request) GetBody() io.Reader {
	if h.IsEmptyBody() {
		return nil
	}
	return h.body.Reader()
}

func (h Request) ToHTTP() (*http.Request, error) {
	h.body.Close()

	req, err := http.NewRequest( 
		strings.ToUpper(strings.TrimSpace((h.Method))),
		h.URL,
		h.GetBody(),
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if h.QueryParams != nil {
		for key, val := range h.QueryParams {
			for _, v := range val {
				q.Add(key, v)
			}
		}
	}
	req.URL.RawQuery = q.Encode()

	if h.Headers != nil {
		for k, vals := range h.Headers {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
	}

	if h.Cookies != nil {
		for _, v := range h.Cookies {
			req.AddCookie(&http.Cookie{Name: v.Name, Value: v.Value})
		}
	}

	if !h.IsEmptyBody() {
		req.Header.Add("Content-Type", h.body.ContentType())
	}

	return req, nil
}

// TODO: set, get, del, remove
func (h *Request) AddQueryParam(key string, val ...string) {
	h.QueryParams[key] = append(h.QueryParams[key], val...)
}

// TODO: set, get, del, remove
func (h *Request) AddHeader(key string, val ...string) {
	h.Headers[key] = append(h.Headers[key], val...)
}

// TODO: set, get, del
// TODO: replace key, val with a whole cookie
func (h *Request) AddCookie(key string, val string) {
	h.Cookies = append(h.Cookies, Cookie{Name: key, Value: val})
}

func (h *Request) SetBody(b Body) {
	h.body = b
}
