package grpc_server

import (
	"context"
	"net"
	"net/http"

	"github.com/sagernet/sing-box/log"
)

// CoreLogger is the log factory built in setupCore; it is forwarded to the
// box instance via boxapi.Create so kernel logs are written to the same
// destination (e.g. neko.log) as the core process.
var CoreLogger log.ObservableFactory

type ProxyCore interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	ListenPacket(ctx context.Context) (net.PacketConn, error)
	CreateProxyHttpClient() *http.Client
	// Shutdown releases core resources (e.g. the running box instance).
	Shutdown() error
}
