package main

import (
	"crypto/tls"
	"net/http"
	"time"
)

var (
	//CoreClient is default http client for Core requests
	CoreClient *http.Client
)

func init() {
	if settings.LOCAL_RESOLVER {
		transportHTTP2 := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: settings.INSECURE_SKIP_VERIFY,
				NextProtos:         []string{"h2"},
				MinVersion:         tls.VersionTLS12,
				Certificates:       []tls.Certificate{credentials.clientKeyPair},
				ClientCAs:          credentials.caCertPool,
			},
		}
		CoreClient = &http.Client{
			Transport: transportHTTP2,
			Timeout:   time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
		}
	} else {
		transportHTTP11 := &http.Transport{
			MaxIdleConnsPerHost: 20,
		}
		CoreClient = &http.Client{
			Transport: transportHTTP11,
			Timeout:   time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
		}
	}
}
