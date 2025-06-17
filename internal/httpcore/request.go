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
			Timeout: 30,
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
			Timeout: 30,
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

	cancel context.CancelFunc
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
	buf := shared.StringBuilder()
	defer shared.PutStringBuilder(buf)

	fullURL := r.URL
	if len(r.QueryParams) > 0 {
		params := url.Values(r.QueryParams)
		fullURL += "?" + params.Encode()
	}
	fmt.Fprintf(buf, "%s %s\n", r.Method, fullURL)

	for name, values := range r.Headers {
		for _, value := range values {
			fmt.Fprintf(buf, "  %s: %s\n", name, value)
		}
	}

	if len(r.Cookies) > 0 {
		var cookiePairs []string
		for _, cookie := range r.Cookies {
			cookiePairs = append(cookiePairs, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
		fmt.Fprintf(buf, "  Cookie: %s\n", strings.Join(cookiePairs, "; "))
	}

	if r.Body != nil && *r.Body.Text != "" {
		fmt.Fprintf(buf, "\n%s\n", *r.Body.Text)
	}

	return buf.String()
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.Options.Timeout) * time.Second)
	h.cancel = cancel

	req, err := http.NewRequestWithContext( // NOTE: shoud i use with context? and why?
		ctx,
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
