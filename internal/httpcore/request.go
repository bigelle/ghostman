package httpcore

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func NewRequest(reqURL string) (*Request, error) {
	_, err := url.ParseRequestURI(reqURL)
	if err != nil {
		return nil, err
	}
	return newRequest(reqURL, "GET")
}

func newRequest(url, method string) (*Request, error) {
	query := make(map[string][]string)
	headers := make(map[string][]string)
	req := Request{
		Method:      method,
		URL:         url,
		QueryParams: &query,
		Headers:     &headers,

		ShouldDumpRequest:   false,
		ShouldDumpResponse:  false,
		ShouldSendRequest:   true,
		ShouldSanitizeQuery: true,
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
	var req Request

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
	Method      string               `json:"method"`
	URL         string               `json:"url"`
	QueryParams *map[string][]string `json:"query_params,omitempty"`
	Headers     *map[string][]string `json:"headers,omitempty"`
	Cookies     *[]Cookie            `json:"cookies,omitempty"`
	Body        *BodySpec            `json:"body,omitempty"`

	// runtime opts
	ShouldDumpRequest     bool `json:"should_dump_request"`
	ShouldDumpResponse    bool `json:"should_dump_response"`
	ShouldSendRequest     bool `json:"should_send_request"`
	ShouldSanitizeQuery   bool `json:"should_sanitize_query"`
	ShouldSanitizeHeaders bool `json:"should_sanitize_headers"`
	ShouldSanitizeCookies bool `json:"should_sanitize_cookies"`

	// only through flags or methods
	body Body
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
	req, err := http.NewRequest( // NOTE: shoud i use with context? and why?
		strings.ToUpper(strings.TrimSpace((h.Method))),
		h.URL,
		h.GetBody(),
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if h.QueryParams != nil {
		for key, val := range *h.QueryParams {
			for _, v := range val {
				q.Add(key, v)
			}
		}
	}
	req.URL.RawQuery = q.Encode()

	if h.Headers != nil {
		for k, vals := range *h.Headers {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
	}

	if *h.Cookies != nil {
		for _, v := range *h.Cookies {
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
	if h.ShouldSanitizeQuery && len(val) == 0 {
		return
	}
	(*h.QueryParams)[key] = append((*h.QueryParams)[key], val...)
}

// TODO: set, get, del, remove
func (h *Request) AddHeader(key string, val ...string) {
	if h.ShouldSanitizeHeaders && len(val) == 0 {
		return
	}
	(*h.Headers)[key] = append((*h.Headers)[key], val...)
}

// TODO: set, get, del
// TODO: replace key, val with a whole cookie
func (h *Request) AddCookie(key string, val string) {
	if h.ShouldSanitizeCookies && len(val) == 0 {
		return
	}
	*h.Cookies = append(*h.Cookies, Cookie{Name: key, Value: val})
}

func (h *Request) SetBody(b Body) {
	h.body = b
}

