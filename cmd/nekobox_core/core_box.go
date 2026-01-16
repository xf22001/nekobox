package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"libneko/neko_common"
	"libneko/neko_log"

	"libneko/boxapi"

	box "github.com/sagernet/sing-box"
	boxmain "github.com/sagernet/sing-box/cmd/sing-box"
)

// InstanceManager 管理 sing-box 实例的创建、访问和销毁
type InstanceManager struct {
	mu     sync.RWMutex
	box    *box.Box
	cancel context.CancelFunc
}

var instanceManager = &InstanceManager{}

// GetInstance 获取当前实例（读锁保护）
func (im *InstanceManager) GetInstance() *box.Box {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.box
}

// SetInstance 设置新实例（写锁保护）
func (im *InstanceManager) SetInstance(b *box.Box, cancel context.CancelFunc) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.box != nil {
		im.box.Close()
	}

	im.box = b
	im.cancel = cancel
}

// ClearInstance 清除当前实例（写锁保护）
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

// GetOrEmpty 返回实例或 nil（读锁保护）
func (im *InstanceManager) GetOrEmpty() (*box.Box, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.box, im.box != nil
}

// ExecuteWithInstance 在实例上执行操作（读锁保护）
func (im *InstanceManager) ExecuteWithInstance(fn func(*box.Box) error) error {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if im.box == nil {
		return errors.New("no running instance available")
	}

	return fn(im.box)
}

func setupCore() {
	boxmain.SetDisableColor(true)
	//
	neko_log.SetupLog(50*1024, "./neko.log")
	//
	neko_common.GetCurrentInstance = func() interface{} {
		return instanceManager.GetInstance()
	}
	neko_common.DialContext = func(ctx context.Context, specifiedInstance interface{}, network, addr string) (net.Conn, error) {
		if i, ok := specifiedInstance.(*box.Box); ok {
			return boxapi.DialContext(ctx, i, network, addr)
		}
		currentInstance := instanceManager.GetInstance()
		if currentInstance != nil {
			return boxapi.DialContext(ctx, currentInstance, network, addr)
		}
		return neko_common.DialContextSystem(ctx, network, addr)
	}
	neko_common.DialUDP = func(ctx context.Context, specifiedInstance interface{}) (net.PacketConn, error) {
		if i, ok := specifiedInstance.(*box.Box); ok {
			return boxapi.DialUDP(ctx, i)
		}
		currentInstance := instanceManager.GetInstance()
		if currentInstance != nil {
			return boxapi.DialUDP(ctx, currentInstance)
		}
		return neko_common.DialUDPSystem(ctx)
	}
	neko_common.CreateProxyHttpClient = func(specifiedInstance interface{}) *http.Client {
		if i, ok := specifiedInstance.(*box.Box); ok {
			return boxapi.CreateProxyHttpClient(i)
		}
		return boxapi.CreateProxyHttpClient(instanceManager.GetInstance())
	}
}
