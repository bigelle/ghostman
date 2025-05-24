package httpcmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type HttpRequest struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Cookies     []Cookie            `json:"cookies"`
	Body        HttpBody            `json:"body"`

	// runtime opts
	ShouldDumpRequest  bool `json:"should_dump_request"`
	ShouldSendRequest  bool `json:"should_send_request"`
	ShouldDumpResponse bool `json:"should_dump_response"`
}

func (h HttpRequest) Request() (*http.Request, error) {
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
	for _, c := range h.Cookies {
		req.AddCookie(c.Http())
	}

	return req, nil
}

func Read(r io.Reader, dest *HttpRequest) error {
	return json.NewDecoder(r).Decode(dest)
}

type HttpBody struct {
	ContentType string `json:"content_type"`
	Body        string `json:"body"` // temporarily just a string
}

// created only to make sure that it has proper json tags
type Cookie struct {
	Name        string    `json:"name"`
	Value       string    `json:"value"`
	Domain      string    `json:"domain"`
	Expires     time.Time `json:"expires"`
	HttpOnly    bool      `json:"http_only"`
	MaxAge      int       `json:"max_age"`
	Partitioned bool      `json:"partitioned"`
	Path        string    `json:"path"`
	SameSite    string    `json:"same_site"` // FIXME: add a enum
	Secure      bool      `json:"secure"`
}

func (c Cookie) Http() *http.Cookie {
	return &http.Cookie{
		Name:        c.Name,
		Value:       c.Value,
		Domain:      c.Domain,
		Expires:     c.Expires,
		HttpOnly:    c.HttpOnly,
		MaxAge:      c.MaxAge,
		Partitioned: c.Partitioned,
		Path:        c.Path,
		SameSite:    http.SameSiteDefaultMode, // FIXME: default by now but should be fixed
		Secure:      c.Secure,
	}
}

var c http.Cookie
