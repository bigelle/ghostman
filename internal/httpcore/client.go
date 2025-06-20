package httpcore

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	client    *http.Client
	transport *http.Transport
	dialer    *net.Dialer
}

type ClientOption func(*http.Client, *http.Transport, *net.Dialer)

func NewClient(opts ...ClientOption) *Client {
	dialer := &net.Dialer{
		Timeout:   8 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
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

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, opt := range opts {
		opt(client, transport, dialer)
	}

	return &Client{client: client, transport: transport, dialer: dialer}
}

func (c *Client) Send(req *RequestConf) (*Response, error) {
	r:= req.ToHTTP()

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	res, err :=  NewResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return res, nil
}
