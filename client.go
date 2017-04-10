package main

import (
	"crypto/tls"
	"net/http"
	"time"
)

var (
	//CoreClient is default http client for Core requests
	CoreClient *http.Client
	//CoreCacheClient http client used for downloading Core cache files
	CoreCacheClient *http.Client
)

func initCoreClient(settings Settings) {
	if settings.LOCAL_RESOLVER {
		tls := &tls.Config{
			InsecureSkipVerify: settings.INSECURE_SKIP_VERIFY,
			MinVersion:         tls.VersionTLS12,
			Certificates:       []tls.Certificate{credentials.clientKeyPair},
			ClientCAs:          credentials.caCertPool,
		}
		CoreClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tls,
			},
			Timeout: time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
		}

		CoreCacheClient = &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
				TLSClientConfig:       tls,
			},
			Timeout: time.Duration(settings.CACHE_REQUEST_TIMEOUT) * time.Second,
		}
	} else {
		transportHTTP11 := &http.Transport{
			MaxIdleConnsPerHost: 20,
		}
		CoreClient = &http.Client{
			Transport: transportHTTP11,
			Timeout:   time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
		}

		CoreCacheClient = &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
			},
			Timeout: time.Duration(settings.CACHE_REQUEST_TIMEOUT) * time.Second,
		}
	}
}
