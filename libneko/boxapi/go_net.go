package boxapi

import (
	"context"
	"net"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing/common/metadata"
)

func DialContext(ctx context.Context, box *box.Box, network, addr string) (net.Conn, error) {
	conn, err := box.Outbound().Default().DialContext(ctx, network, metadata.ParseSocksaddr(addr))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func DialUDP(ctx context.Context, box *box.Box) (net.PacketConn, error) {
	return box.Outbound().Default().ListenPacket(ctx, metadata.Socksaddr{})
}
