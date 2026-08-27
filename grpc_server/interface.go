package grpc_server

import (
	"context"
	"net"
	"net/http"
)

// ProxyCore is the engine facade implemented by the core process. The gRPC
// layer only depends on this interface so the underlying engine (sing-box /
// Xray) can be swapped without touching the service wiring.
type ProxyCore interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	ListenPacket(ctx context.Context) (net.PacketConn, error)
	CreateProxyHttpClient() *http.Client
	// Shutdown releases core resources (e.g. the running engine instance).
	Shutdown() error
}
