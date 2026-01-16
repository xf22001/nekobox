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
		TLSHandshakeTimeout:   time.Second * 5,      // 增加 TLS 握手超时
		ResponseHeaderTimeout: time.Second * 10,     // 增加响应头超时
		ExpectContinueTimeout: time.Second * 1,      // 保持期望继续超时
		IdleConnTimeout:       time.Second * 30,     // 增加空闲连接超时
		MaxIdleConns:          100,                  // 保持合理的最大空闲连接数
		MaxIdleConnsPerHost:   10,                   // 保持每主机最大空闲连接数
	}

	if box != nil {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return DialContext(ctx, box, network, addr)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 60,                 // 增加总超时时间
	}

	return client
}
