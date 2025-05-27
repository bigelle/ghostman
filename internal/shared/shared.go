package shared

import (
	"crypto/tls"
	"net/http"
	"sync"
	"time"
)

//TODO: should be configured on init
var HttpClientPool = sync.Pool{
	New: func() any {
		return &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxConnsPerHost:     10,
				MaxIdleConnsPerHost: 10,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				Proxy:               http.ProxyFromEnvironment,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
	},
}

