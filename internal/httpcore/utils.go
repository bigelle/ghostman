package httpcore

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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

func DetectSchema(rawURL string) (fullURL string, err error) {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL, nil
	}

	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	// trying HTTPS:
	req, err := http.NewRequest("HEAD", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("doing HEAD request to detect HTTP/S schema: %w", err)
	}

	_, err = client.Do(req)
	if err == nil {
		return rawURL, nil
	}

	// trying HTTP
	rawURL = strings.ReplaceAll(rawURL, "https://", "http://")
	req, err = http.NewRequest("HEAD", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("doing HEAD request to detect HTTP/S: %w", err)
	}

	_, err = client.Do(req)
	if err == nil {
		return rawURL, nil
	}

	return "", fmt.Errorf("unable to detect schema for this url")
}
