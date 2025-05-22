package httpcmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func parseHTTPHeaders(h []string) (*map[string][]string, error) {
	// example: -H "Accept:application/json,text/plain"
	// should return: "Accept": {"application/json", "text/plain"}
	headers := make(map[string][]string)
	for _, raw := range h {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Wrong header format: %s\n", raw)
		}
		key := strings.TrimSpace(parts[0])
		values := strings.Split(parts[1], ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		headers[key] = append(headers[key], values...)
	}
	return &headers, nil
}

func setupHeaders(req *HttpRequest) error {
	hs, err := parseHTTPHeaders(headers)
	if err != nil {
		return err
	}

	for k, val := range *hs {
		for _, v := range val {
			req.Headers[k] = append(req.Headers[k], v)
		}
	}

	return nil
}

func parseCommand(cmd *cobra.Command, args []string) error {
	httpRequest = HttpRequest{
		// Method would be set during the RunE func
		URL: args[0],
		//QueryParams are probably already in url
		//NOTE: or should i add a flag to add query params?
		//
		//anyway
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),

		ShouldDumpRequest:  shouldDumpRequest,
		ShouldSendRequest:  shouldSendRequest,
		ShouldDumpResponse: shouldDumpResponse,
	}

	host, err := extractHost(args[0])
	if err != nil {
		return err
	}
	httpRequest.Headers["Host"] = append(httpRequest.Headers["Host"], *host)

	if err := setupHeaders(&httpRequest); err != nil {
		return err
	}

	return nil
}

func extractHost(rawURL string) (*string, error) {
	p, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := p.Hostname()
	return &host, nil
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes)) 
	}

	clone := req.Clone(req.Context())
	if bodyBytes != nil {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	return clone, nil
}

func dumpRequestSafely(req *http.Request) ([]byte, error) {
	clone, err := cloneRequest(req)
	if err != nil {
		return nil, err
	}
	return httputil.DumpRequestOut(clone, req.Body != nil)
}
