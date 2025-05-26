package httpcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// works with both headers and query parameters
func parseHTTPKeyValues(h []string) (*map[string][]string, error) {
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

func parseCookieKeyValue(c []string) (*map[string]string, error) {
	result := make(map[string]string)
	for _, raw := range c {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Wrong key:value pair format: %s\n", raw)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}
	return &result, nil
}

func parseCommand(cmd *cobra.Command, args []string) error {
	req := HttpRequest{
		// Method would be set during the RunE func
		Method:      cmd.Use,
		URL:         args[0],
		QueryParams: make(map[string][]string),
		Headers:     make(map[string][]string),
	}
	dumpReq, _ := cmd.Flags().GetBool("dump-request")
	if dumpReq != req.ShouldDumpRequest {
		req.ShouldSendRequest = dumpReq
	}
	dumpResp, _ := cmd.Flags().GetBool("dump-response")
	if dumpResp != req.ShouldDumpResponse{
		req.ShouldDumpResponse = dumpResp
	}
	sendReq, _ := cmd.Flags().GetBool("send-request")
	if sendReq != req.ShouldSendRequest{
		req.ShouldSendRequest = sendReq
	}

	host, err := extractHost(args[0])
	if err != nil {
		return err
	}
	req.Headers["Host"] = append(req.Headers["Host"], host)

	jsonBody, _ := cmd.Flags().GetString("data-json")
	if jsonBody != "" {
		if err := json.NewDecoder(strings.NewReader(jsonBody)).Decode(&map[string]any{}); err != nil {
			return err
		}
		req.Body = HttpBody{
			ContentType: "application/json",
			Body:        jsonBody,
		}
	}


	headers, _ := cmd.Flags().GetStringArray("header")
	hs, err := parseHTTPKeyValues(headers)
	if err != nil {
		return err
	}

	for k, val := range *hs {
		for _, v := range val {
			req.Headers[k] = append(req.Headers[k], v)
		}
	}

	cookies, _ := cmd.Flags().GetStringArray("cookie")
	parsed, err := parseCookieKeyValue(cookies)
	if err != nil {
		return err
	}
	req.Cookies = *parsed

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
	buf := bytes.NewBuffer([]byte{})
	if _, err := buf.ReadFrom(read); err != nil {
		return err
	}

	req, err := NewHttpRequestFromJSON(buf.Bytes()); if err != nil {
		return err
	}

	dumpReq, _ := cmd.Flags().GetBool("dump-request")
	if dumpReq != req.ShouldDumpRequest {
		req.ShouldSendRequest = dumpReq
	}
	dumpResp, _ := cmd.Flags().GetBool("dump-response")
	if dumpResp != req.ShouldDumpResponse{
		req.ShouldDumpResponse = dumpResp
	}
	sendReq, _ := cmd.Flags().GetBool("send-request")
	if sendReq != req.ShouldSendRequest{
		req.ShouldSendRequest = sendReq
	}

	ctx := cmd.Context()
	withVal := context.WithValue(ctx, "httpReq", req)
	cmd.SetContext(withVal)
	return nil
}

func extractHost(rawURL string) (string, error) {
	p, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := p.Hostname()
	return host, nil
}


func extractQueryParams(raw string) (map[string][]string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return parsed.Query(), nil
}
