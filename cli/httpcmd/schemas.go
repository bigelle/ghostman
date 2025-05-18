package httpcmd

import (
	"net/http"
	"strings"
)

type HttpRequest struct {
	Method      string
	URL         string
	QueryParams map[string][]string
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

	return req, nil
}
