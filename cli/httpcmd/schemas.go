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

func Read(r io.Reader, dest *HttpRequest) error {
	return json.NewDecoder(r).Decode(dest)
}
