package boxapi

import (
	"context"
	"net"
	"net/http"
	"time"

	box "github.com/sagernet/sing-box"
)

func CreateProxyHttpClient(box *box.Box) *http.Client {
	transport := &http.Transport{
		TLSHandshakeTimeout:   time.Second * 3,
		ResponseHeaderTimeout: time.Second * 3,
		DialTLSContext: (&net.Dialer{
			Timeout:   time.Second * 3,
			KeepAlive: time.Second * 5,
		}).DialContext,
		ExpectContinueTimeout: time.Second * 1,
		IdleConnTimeout:       time.Second * 5,
		MaxIdleConns:          1,
		MaxIdleConnsPerHost:   1,
	}

	if box != nil {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return DialContext(ctx, box, network, addr)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 10, // Default timeout for the entire request
	}

	return client
}
