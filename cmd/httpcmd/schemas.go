package httpcmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type HttpRequest struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Body        HttpBody            `json:"body"`

	// runtime opts
	ShouldDumpRequest  bool
	ShouldSendRequest  bool
	ShouldDumpResponse bool
}

type HttpBody struct {
	ContentType string `json:"content_type"`
	Body        string `json:"body"` // temporarily just a string
}

func (h HttpRequest) Request() (*http.Request, error) {
	req, err := http.NewRequest(
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

	return req, nil
}

func Read(r io.Reader, dest *HttpRequest) error {
	return json.NewDecoder(r).Decode(dest)
}
