package httpcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss/tree"
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
	req  *http.Request
}

type Options struct {
	Verbose         *bool `json:"verbose,omitempty"`
	SendRequest     *bool `json:"send_request,omitempty"`
	SanitizeQuery   *bool `json:"sanitize_query,omitempty"`
	SanitizeHeaders *bool `json:"sanitize_headers,omitempty"`
	SanitizeCookies *bool `json:"sanitize_cookies,omitempty"`
	Timeout         int   `json:"timeout,omitempty"`
}

func (r Request) ToString() (string, error) {
	if r.req == nil {
		r.ToHTTP()
	}

	dump, err := DumpRequest(r.req)
	if err != nil {
		return "", err
	}

	bodyIndex := bytes.Index(dump, []byte("\r\n\r\n"))
	var body []byte
	if bodyIndex != -1 {
		body = dump[bodyIndex+4:]
	}

	rows := strings.Split(string(dump), "\r\n")
	if len(rows) == 0 {
		return "", fmt.Errorf("malformed request")
	}

	rows = rows[1:]

	var headers []string
	var cookies []string
	for _, row := range rows {
		if strings.TrimSpace(row) == "" {
			break
		}
		if strings.HasPrefix(row, "Cookie:") {
			// temporarily just print them without any formatting
			cookies = append(cookies, strings.TrimSpace(strings.TrimPrefix(row, "Cookie:")))
			continue
		}
		headers = append(headers, row)
	}

	t := tree.Root(fmt.Sprintf("%s %s", r.Method, r.URL)) // temporarily without query

	if len(headers) > 0 {
		h := tree.Root("Headers:")
		for _, header := range headers {
			h.Child(header)
		}
		t.Child(h)
	}

	if len(cookies) > 0 {
		c := tree.Root("Cookie:")
		for _, cookie := range cookies {
			c.Child(cookie)
		}
		t.Child(c)
	}

	if body != nil {
		// temporarily just print the length
		t.Child(fmt.Sprintf("Body: %d bytes", len(body)))
	}

	return t.String(), nil
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

func (r *Request) ToHTTP() (*http.Request, error) {
	r.body.Close()

	req, err := http.NewRequest(
		strings.ToUpper(strings.TrimSpace((r.Method))),
		r.URL,
		r.GetBody(),
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if r.QueryParams != nil {
		for key, val := range r.QueryParams {
			for _, v := range val {
				q.Add(key, v)
			}
		}
	}
	req.URL.RawQuery = q.Encode()

	if r.Headers != nil {
		for k, vals := range r.Headers {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
	}

	if r.Cookies != nil {
		for _, v := range r.Cookies {
			req.AddCookie(&http.Cookie{Name: v.Name, Value: v.Value})
		}
	}

	if !r.IsEmptyBody() {
		req.Header.Add("Content-Type", r.body.ContentType())
	}

	r.req = req
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
