package grpc_server

import (
	"context"
	"net"
	"net/http"
)

type ProxyCore interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	ListenPacket(ctx context.Context) (net.PacketConn, error)
	CreateProxyHttpClient() *http.Client
	// Shutdown releases core resources (e.g. the running box instance).
	Shutdown() error
}
