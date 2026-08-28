package main

import (
	"time"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/xrayapi"
)

// InstanceManager holds the single running Xray instance, mirroring the
// sing-box based design so the gRPC layer above stays unchanged.
type InstanceManager struct {
	mu     sync.RWMutex
	core   *core.Instance
	cancel context.CancelFunc
}

var instanceManager = &InstanceManager{}

func (im *InstanceManager) GetInstance() *core.Instance {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.core
}

func (im *InstanceManager) SetInstance(c *core.Instance, cancel context.CancelFunc) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.core != nil {
		_ = im.core.Close()
		time.Sleep(200 * time.Millisecond)
	}
	im.core = c
	im.cancel = cancel
	fmt.Println("Core instance updated and started")
}

func (im *InstanceManager) ClearInstance() {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.cancel != nil {
		im.cancel()
	}
	if im.core != nil {
		_ = im.core.Close()
		im.core = nil
		time.Sleep(300 * time.Millisecond)
	}
	im.cancel = nil
}

func (im *InstanceManager) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return xrayapi.DialContext(im.GetInstance(), ctx, network, addr)
}

func (im *InstanceManager) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	// Xray routes UDP through the dispatcher as a connection; for the packet
	// oriented use case a plain UDP socket is sufficient.
	return net.ListenUDP("udp", &net.UDPAddr{})
}

func (im *InstanceManager) CreateProxyHttpClient() *http.Client {
	return xrayapi.CreateHttpClient(im.GetInstance())
}

func setupCore() {
	fmt.Println("Nekoray core initialized (Xray engine)")
	if logFile, err := os.OpenFile("neko.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		logFile.Close()
	}
}
