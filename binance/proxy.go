package binance

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ProxyRoundTripper sets the proxy auth header on the request before making the reqeust.
func ProxyRoundTripper(original *http.Transport, username, password string) http.RoundTripper {
	if original == nil {
		basicHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
		original = &http.Transport{
			TLSClientConfig:    &tls.Config{},
			ProxyConnectHeader: http.Header{"Proxy-Authorization": []string{"Basic " + basicHeader}},
			Proxy:              http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	auth := username + ":" + password
	base64Auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	return roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		request.Header.Set("Proxy-Authorization", base64Auth)

		response, err := original.RoundTrip(request)
		return response, err
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
