package httpcore

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	client *http.Client
	once   sync.Once
)

func Client() *http.Client {
	once.Do(func() {
		client = basicClient() // without options for now
	})
	return client
}

func basicClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:   false,
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 50,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     60 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   8 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS11,
				MaxVersion:         tls.VersionTLS13,
			},
			DisableCompression: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 60 * time.Second,
	}
}

func Send(req *Request) (*Response, error) {
	if client == nil {
		client = basicClient()
	}

	r, err := req.ToHTTP()
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return NewResponse(resp)
}
