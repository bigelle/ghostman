package httpcore

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func ExtractQueryParams(raw string) (map[string][]string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return parsed.Query(), nil
}

func DumpRequest(req *http.Request) (dump []byte, err error) {
	var buf []byte

	if req.Body != nil {
		buf, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(buf))
	}

	dump, err = httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(buf))

	return dump, err
}
