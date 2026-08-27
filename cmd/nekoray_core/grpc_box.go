package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"nekoray/grpc_server"
	"nekoray/grpc_server/gen"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/xrayapi"
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

func (s *server) Shutdown() error {
	instanceManager.ClearInstance()
	return nil
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
		fmt.Println("Start with config: ", in.CoreConfig)
	}

	if instanceManager.GetInstance() != nil {
		return &gen.ErrorResp{Error: "instance already started"}, nil
	}

	instance, err := xrayapi.Create([]byte(in.CoreConfig))
	if err != nil {
		return &gen.ErrorResp{Error: err.Error()}, nil
	}

	instanceManager.SetInstance(instance, nil)
	return &gen.ErrorResp{}, nil
}

func (s *server) Stop(ctx context.Context, in *gen.EmptyReq) (out *gen.ErrorResp, _ error) {
	instanceManager.ClearInstance()
	return &gen.ErrorResp{}, nil
}

func (s *server) Test(ctx context.Context, in *gen.TestReq) (out *gen.TestResp, _ error) {
	var err error
	out = &gen.TestResp{Ms: 0}

	defer func() {
		if err != nil {
			out.Error = err.Error()
		}
	}()

	instance, cleanup, err := s.getOrCreateInstance(in.Config)
	if err != nil {
		return &gen.TestResp{Error: err.Error()}, nil
	}
	if cleanup != nil {
		defer cleanup()
	}

	switch in.Mode {
	case gen.TestMode_UrlTest:
		if instance == nil {
			return out, nil
		}
		client := xrayapi.CreateHttpClient(instance)
		out.Ms, err = grpc_server.UrlTest(client, in.Url, in.Timeout, grpc_server.UrlTestStandard_RTT)

	case gen.TestMode_TcpPing:
		out.Ms, err = grpc_server.TcpPing(in.Address, in.Timeout)

	case gen.TestMode_FullTest:
		// FullTest 现在通过 ProxyCore 接口直接交互
		return grpc_server.DoFullTest(ctx, in, s)

	case gen.TestMode_CheckProxy:
		if instance == nil {
			out.Error = "no instance available"
			return
		}
		client := xrayapi.CreateHttpClient(instance)
		fetchTimeout := time.Duration(in.Timeout) * time.Millisecond
		if fetchTimeout == 0 {
			fetchTimeout = 10 * time.Second
		}
		client.Timeout = fetchTimeout

		info, ipInfoErr := FetchIPInfo(ctx, client)
		if ipInfoErr != nil {
			out.Error = "IP info fetch failed: " + ipInfoErr.Error()
			return
		}

		var parts []string
		if info.Country != "" {
			parts = append(parts, info.Country)
		}
		if info.City != "" {
			parts = append(parts, info.City)
		}
		if info.Isp != "" {
			parts = append(parts, info.Isp)
		}

		location := ""
		if len(parts) > 0 {
			location = " (" + strings.Join(parts, ", ") + ")"
		}

		fmt.Println("IP Info: ", info.Query, location)
		out.FullReport = info.Query + location
	}

	return
}

func (s *server) QueryStats(ctx context.Context, in *gen.QueryStatsReq) (*gen.QueryStatsResp, error) {
	instance := instanceManager.GetInstance()
	if instance == nil {
		return &gen.QueryStatsResp{Traffic: 0}, nil
	}

	tag := in.GetTag()
	if tag == "" {
		tag = "proxy"
	}
	direct := in.GetDirect()
	if direct == "" {
		direct = "uplink"
	}
	name := fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", tag, direct)
	// reset=true so pynekoray can treat each poll as speed sample
	traffic, err := xrayapi.GetNekoStats(instance, name, true)
	if err != nil {
		// missing counter means no traffic yet
		return &gen.QueryStatsResp{Traffic: 0}, nil
	}
	return &gen.QueryStatsResp{Traffic: traffic}, nil
}

func (s *server) getOrCreateInstance(config *gen.LoadConfigReq) (*core.Instance, func(), error) {
	if config != nil {
		instance, err := xrayapi.Create([]byte(config.CoreConfig))
		if err != nil {
			return nil, nil, err
		}
		if instance == nil {
			return nil, nil, fmt.Errorf("instance creation failed")
		}
		cleanup := func() {
			_ = instance.Close()
		}
		return instance, cleanup, nil
	}

	instance := instanceManager.GetInstance()
	if instance == nil {
		return nil, nil, fmt.Errorf("no instance available")
	}
	return instance, nil, nil
}
