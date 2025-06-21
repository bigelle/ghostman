package httpcore

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func ExtractQueryParams(raw string) (map[string][]string, error) {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return nil, fmt.Errorf("error parsing query: %w", err)
	}
	return parsed.Query(), nil
}

func DumpRequest(req *http.Request) (dump []byte, err error) {
	var buf []byte

	if req.Body != nil {
		buf, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(buf))
	}

	dump, err = httputil.DumpRequestOut(req, true)

	req.Body = io.NopCloser(bytes.NewReader(buf))

	if err != nil {
		return nil, fmt.Errorf("dumping request: %w", err)
	}

	return dump, nil
}

func DumpResponse(resp *http.Response, body bool) (dump []byte, err error) {
	var buf []byte

	if resp.Body != nil {
		buf, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}
		resp.Body = io.NopCloser(bytes.NewReader(buf))
	}

	dump, err = httputil.DumpResponse(resp, body)
	
	resp.Body = io.NopCloser(bytes.NewReader(buf))

	if err != nil {
		return nil, fmt.Errorf("dumping response: %w", err)
	}

	return dump, nil
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

type BytesReadCloser struct {
	bytes.Reader
}

func (b BytesReadCloser) Close() error {
	return nil
}
