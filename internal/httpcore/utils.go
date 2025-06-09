package httpcore

import (
	"net/url"
)

func ExtractQueryParams(raw string) (map[string][]string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return parsed.Query(), nil
}
