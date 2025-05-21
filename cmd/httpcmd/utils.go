package httpcmd

import (
	"fmt"
	"net/http"
	"strings"
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

func setupHeaders(req *http.Request) error {
	hs, err := parseHTTPHeaders(headers)
	if err != nil {
		return err
	}

	for k, val := range *hs {
		for _, v := range val {
			req.Header.Add(k,v)
		}
	}

	return nil
}
