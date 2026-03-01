package main

import (
	"net"
	"net/http"
	"context"
	"errors"
	"time"

	"nekobox/grpc_server"
	"nekobox/grpc_server/gen"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/boxapi"
	"github.com/sagernet/sing-box/log"
)

type server struct {
	grpc_server.BaseServer
}

// 确保 server 实现了 grpc_server.ProxyCore 接口
var _ grpc_server.ProxyCore = (*server)(nil)

func (s *server) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return instanceManager.DialContext(ctx, network, addr)
}

func (s *server) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	return instanceManager.ListenPacket(ctx)
}

func (s *server) CreateProxyHttpClient() *http.Client {
	return instanceManager.CreateProxyHttpClient()
}

func (s *server) Start(ctx context.Context, in *gen.LoadConfigReq) (out *gen.ErrorResp, _ error) {
	var err error

	defer func() {
		out = &gen.ErrorResp{}
		if err != nil {
			out.Error = err.Error()
		}
	}()

	if grpc_server.Debug {
		log.Info("Start with config: ", in.CoreConfig)
	}

	currentInstance := instanceManager.GetInstance()
	if currentInstance != nil {
		return &gen.ErrorResp{Error: "instance already started"}, nil
	}

	newInstance, newCancel, err := boxapi.Create([]byte(in.CoreConfig), nil)
	if err != nil {
		return &gen.ErrorResp{Error: err.Error()}, nil
	}

	if newInstance != nil {
		instanceManager.SetInstance(newInstance, newCancel)
	} else {
		log.Error("err: ", err)
		err = errors.New("failed to create instance")
	}

	return &gen.ErrorResp{}, nil
}

func (s *server) Stop(ctx context.Context, in *gen.EmptyReq) (out *gen.ErrorResp, _ error) {
	var err error

	defer func() {
		out = &gen.ErrorResp{}
		if err != nil {
			out.Error = err.Error()
		}
	}()

	instanceManager.ClearInstance()
	return
}

func (s *server) Test(ctx context.Context, in *gen.TestReq) (out *gen.TestResp, _ error) {
	var err error
	out = &gen.TestResp{Ms: 0}

	defer func() {
		if err != nil {
			out.Error = err.Error()
		}
	}()

	switch in.Mode {
	case gen.TestMode_UrlTest:
		i, cleanup, err := s.getOrCreateInstance(in.Config)
		if err != nil {
			return &gen.TestResp{Error: err.Error()}, nil
		}
		if cleanup != nil {
			defer cleanup()
		}
		if i == nil {
			return out, nil
		}
		// 使用重构后的 testing 包和 instanceManager
		client := instanceManager.CreateProxyHttpClient()
		out.Ms, err = grpc_server.UrlTest(client, in.Url, in.Timeout, grpc_server.UrlTestStandard_RTT)

	case gen.TestMode_TcpPing:
		out.Ms, err = grpc_server.TcpPing(in.Address, in.Timeout)

	case gen.TestMode_FullTest:
		// FullTest 现在通过 ProxyCore 接口直接交互
		return grpc_server.DoFullTest(ctx, in, s)

	case gen.TestMode_CheckProxy:
		client := instanceManager.CreateProxyHttpClient()
		fetchTimeout := time.Duration(in.Timeout) * time.Millisecond
		if fetchTimeout == 0 {
			fetchTimeout = 10 * time.Second
		}
		client.Timeout = fetchTimeout

		info, ipInfoErr := FetchIPInfo(ctx, client)
		if ipInfoErr != nil {
			out.Error = "IP info fetch failed"
			return
		}
		log.Info("IP Info: ", info.Query, " (", info.Country, ", ", info.City, ")")
		out.FullReport = info.Query
	}

	return
}

func (s *server) getOrCreateInstance(config *gen.LoadConfigReq) (*box.Box, func(), error) {
	if config != nil {
		i, cancel, err := boxapi.Create([]byte(config.CoreConfig), nil)
		if err != nil {
			return nil, nil, err
		}
		if i == nil {
			return nil, nil, errors.New("instance creation failed")
		}
		cleanup := func() {
			cancel()
			i.Close()
		}
		return i, cleanup, nil
	}

	i := instanceManager.GetInstance()
	if i == nil {
		return nil, nil, errors.New("no instance available")
	}
	return i, nil, nil
}
