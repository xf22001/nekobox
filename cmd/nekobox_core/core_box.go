package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"


	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/boxapi"
	"github.com/sagernet/sing/common/metadata"
)

type InstanceManager struct {
	mu     sync.RWMutex
	box    *box.Box
	cancel context.CancelFunc
}

var instanceManager = &InstanceManager{}

func (im *InstanceManager) GetInstance() *box.Box {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.box
}

func (im *InstanceManager) SetInstance(b *box.Box, cancel context.CancelFunc) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.box != nil {
		im.box.Close()
	}
	im.box = b
	im.cancel = cancel
}

func (im *InstanceManager) ClearInstance() {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.cancel != nil {
		im.cancel()
	}
	if im.box != nil {
		im.box.Close()
	}
	im.box = nil
	im.cancel = nil
}

func (im *InstanceManager) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	instance := im.GetInstance()
	if instance != nil {
		return instance.Outbound().Default().DialContext(ctx, network, metadata.ParseSocksaddr(addr))
	}
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func (im *InstanceManager) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	instance := im.GetInstance()
	if instance != nil {
		return instance.Outbound().Default().ListenPacket(ctx, metadata.Socksaddr{})
	}
	return net.ListenUDP("udp", &net.UDPAddr{})
}

func (im *InstanceManager) CreateProxyHttpClient() *http.Client {
	transport := &http.Transport{
		TLSHandshakeTimeout:   time.Second * 5,
		ResponseHeaderTimeout: time.Second * 10,
		IdleConnTimeout:       time.Second * 30,
	}
	transport.DialContext = im.DialContext
	return &http.Client{
		Transport: transport,
		Timeout:   time.Second * 60,
	}
}

func setupCore() {
	boxapi.SetDisableColor(true)
}
