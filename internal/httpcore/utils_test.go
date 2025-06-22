package httpcore

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDetectProtocolWithMock(t *testing.T) {
	testcases := []struct {
		Name           string
		Input          string
		Expected       string
		ExpectErr      bool
		HTTPSResponse  bool 
		HTTPResponse   bool 
	}{
		{
			Name:          "URL with HTTPS schema",
			Input:         "https://example.com",
			Expected:      "https://example.com",
			HTTPSResponse: true,
		},
		{
			Name:         "URL with HTTP schema",
			Input:        "http://example.com",
			Expected:     "http://example.com",
			HTTPResponse: true,
		},
		{
			Name:          "HTTPS URL without schema",
			Input:         "secure.example.com",
			Expected:      "https://secure.example.com",
			HTTPSResponse: true,
		},
		{
			Name:         "HTTP URL without schema",
			Input:        "insecure.example.com",
			Expected:     "http://insecure.example.com",
			HTTPResponse: true,
		},
		{
			Name:      "No schema, non-existing host",
			Input:     "your.momma",
			Expected:  "",
			ExpectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			client = &http.Client{
				Transport: &mockTransport{
					httpsSuccess: tc.HTTPSResponse,
					httpSuccess:  tc.HTTPResponse,
				},
				Timeout: 5 * time.Second,
			}

			result, err := DetectSchema(tc.Input)

			if tc.ExpectErr {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, result)
			}
		})
	}
}

type mockTransport struct {
	httpsSuccess bool
	httpSuccess  bool
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "https" && m.httpsSuccess {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	if req.URL.Scheme == "http" && m.httpSuccess {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	return nil, errors.New("connection failed")
}
