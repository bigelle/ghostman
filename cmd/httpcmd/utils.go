package httpcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// works with both headers and query parameters
func parseHTTPKeyValue(h []string) (*map[string][]string, error) {
	// example: -H "Accept:application/json,text/plain"
	// should return: "Accept": {"application/json", "text/plain"}
	// same with query params
	result := make(map[string][]string)
	for _, raw := range h {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Wrong key:value pair format: %s\n", raw)
		}
		key := strings.TrimSpace(parts[0])
		values := strings.Split(parts[1], ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		result[key] = append(result[key], values...)
	}
	return &result, nil
}

func setupHeaders(req *HttpRequest) error {
	hs, err := parseHTTPKeyValue(headers)
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
	req := HttpRequest{
		// Method would be set during the RunE func
		Method:      cmd.Use,
		URL:         args[0],
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
	req.Headers["Host"] = append(req.Headers["Host"], *host)

	if jsonBody != "" {
		if err := json.NewDecoder(strings.NewReader(jsonBody)).Decode(&map[string]any{}); err != nil {
			return err
		}
		req.Body = HttpBody{
			ContentType: "application/json",
			Body:        jsonBody,
		}
	}

	if err := setupHeaders(&req); err != nil {
		return err
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, "httpReq", req)
	cmd.SetContext(withVal)

	return nil
}

func readHttpFile(cmd *cobra.Command, args []string) error {
	read, err := os.Open(args[0])
	if err != nil {
		return err
	}
	var req HttpRequest
	if err = Read(read, &req); err != nil {
		return err
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, "httpReq", req)
	cmd.SetContext(withVal)
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
