package httpcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

type Method string

func (m Method) Color() string {
	baseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("FFFFFF")).
		Bold(true).
		Padding(0, 1)
	switch m {
	case "GET":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#16a34a",
				ANSI256:   "28",
				ANSI:      "2",
			},
		)
		return style.Render(string(m))
	case "POST":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#3b82f6",
				ANSI256:   "33",
				ANSI:      "4",
			},
		)
		return style.Render(string(m))
	case "PUT":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f59e0b",
				ANSI256:   "214",
				ANSI:      "3",
			},
		)
		return style.Render(string(m))
	case "PATCH":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#8b5cf6",
				ANSI256:   "99",
				ANSI:      "5",
			},
		)
		return style.Render(string(m))
	case "DELETE":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#ef4444",
				ANSI256:   "196",
				ANSI:      "1",
			},
		)
		return style.Render(string(m))
	case "HEAD":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#06b6d4",
				ANSI256:   "31",
				ANSI:      "6",
			},
		)
		return style.Render(string(m))
	case "OPTIONS":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#84cc16",
				ANSI256:   "112",
				ANSI:      "10",
			},
		)
		return style.Render(string(m))
	case "TRACE":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#64748b",
				ANSI256:   "244",
				ANSI:      "8",
			},
		)
		return style.Render(string(m))
	case "CONNECT":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f97316",
				ANSI256:   "202",
				ANSI:      "9",
			},
		)
		return style.Render(string(m))
	default:
		return baseStyle.Render(string(m))
	}
}

func NewRequest(reqUrl string) (*Request, error) {
	req := Request{
		Method:      http.MethodGet,
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

	_, err := url.ParseRequestURI(reqUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid request URL: %w", err)
	}

	return &req, nil
}

func NewRequestFromJSON(j []byte) (*Request, error) {
	req := Request{
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),
		Options: Options{
			Verbose:         func(b bool) *bool { return &b }(false),
			Out:             "stdout",
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
			return nil, fmt.Errorf("can't attach a body: %w", err)
		}

		req.SetBody(body)
	}

	return &req, nil
}

type Request struct {
	// serializable
	Method      Method              `json:"method"`
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
	Verbose         *bool  `json:"verbose,omitempty"`
	SendRequest     *bool  `json:"send_request,omitempty"`
	SanitizeQuery   *bool  `json:"sanitize_query,omitempty"`
	SanitizeHeaders *bool  `json:"sanitize_headers,omitempty"`
	SanitizeCookies *bool  `json:"sanitize_cookies,omitempty"`
	Timeout         int    `json:"timeout,omitempty"`
	Out             string `json:"out,omitempty"`
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
			// for now just raw key=value
			cookies = append(cookies, strings.TrimSpace(strings.TrimPrefix(row, "Cookie:")))
			continue
		}
		headers = append(headers, row)
	}

	t := tree.Root(fmt.Sprintf("%s %s", r.Method.Color(), r.req.URL))

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
		size := FormatBytes(int64(len(body)))
		t.Child(fmt.Sprintf("Body: %s of %s", size, r.req.Header.Get("Content-Type")))
	}

	return t.String(), nil
}

func (h Request) IsEmptyBody() bool {
	return h.body == nil || h.body.Len() == 0
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
		strings.ToUpper(strings.TrimSpace(string(r.Method))),
		r.URL,
		r.GetBody(),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
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
