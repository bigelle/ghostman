package httpcore

import (
	"fmt"
	"net/url"
	"strings"
)

// works with both headers and query parameters
func ParseKeyValues(h []string) (map[string][]string, error) {
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
	return result, nil
}

func ExtractQueryParams(raw string) (map[string][]string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return parsed.Query(), nil
}

func ParseKeySingleValue(h []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, raw := range h {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Wrong key:value pair format: %s\n", raw)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}
	return result, nil
}
