package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"nekobox/grpc_server"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/boxapi"
	"github.com/sagernet/sing-box/log"
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
	log.Info("Core instance updated and started")
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

	// 恢复功能：日志双写。同时输出到 Stderr 和 neko.log 文件
	var writers []io.Writer
	writers = append(writers, os.Stderr)

	// 显式创建日志文件
	logFile, err := os.OpenFile("neko.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		writers = append(writers, logFile)
	}

	multiWriter := io.MultiWriter(writers...)

	factory := log.NewDefaultFactory(
		context.Background(),
		log.Formatter{DisableColors: true, TimestampFormat: "-0700 2006-01-02 15:04:05", FullTimestamp: true},
		multiWriter,
		"",
		nil,
		false,
	)

	// 根据 -debug 参数设置日志等级
	if grpc_server.Debug {
		factory.SetLevel(log.LevelDebug)
	} else {
		factory.SetLevel(log.LevelInfo)
	}

	err = factory.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start log factory: %v\n", err)
	}
	log.SetStdLogger(factory.Logger())
	
	log.Info("Nekobox core logger initialized")
}
