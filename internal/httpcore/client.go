package httpcore

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var (
	dialer *net.Dialer = &net.Dialer{
		Timeout:   8 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport *http.Transport = &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       60 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
		},
	}

	client *http.Client = &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

type Client struct {
	client    *http.Client
	transport *http.Transport
	dialer    *net.Dialer
}

type ClientOption func(*http.Client, *http.Transport, *net.Dialer)

func NewClient(opts ...ClientOption) *Client {
	for _, opt := range opts {
		opt(client, transport, dialer)
	}

	return &Client{client: client, transport: transport, dialer: dialer}
}

func (c *Client) Send(req *RequestConf) (*Response, error) {
	r := req.ToHTTP()

	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	res := Response{resp: resp}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		// how do i handle this? i still should return a response, but with no body
		return &res, fmt.Errorf("copying response body: %w", err)
	}

	res.body = data

	return &res, nil
}
