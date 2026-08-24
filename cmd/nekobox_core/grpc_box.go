package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"nekobox/grpc_server"
	"nekobox/grpc_server/gen"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/boxapi"
	"github.com/sagernet/sing-box/experimental/clashapi"
	"github.com/sagernet/sing-box/experimental/v2rayapi"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/service"
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
		log.Info("Start with config: ", in.CoreConfig)
	}

	currentInstance := instanceManager.GetInstance()
	if currentInstance != nil {
		return &gen.ErrorResp{Error: "instance already started"}, nil
	}

	newInstance, newCancel, err := boxapi.Create([]byte(in.CoreConfig), grpc_server.CoreLogger)
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

	i, cleanup, err := s.getOrCreateInstance(in.Config)
	if err != nil {
		return &gen.TestResp{Error: err.Error()}, nil
	}
	if cleanup != nil {
		defer cleanup()
	}

	switch in.Mode {
	case gen.TestMode_UrlTest:
		if i == nil {
			return out, nil
		}
		client := CreateHttpClientForBox(i)
		out.Ms, err = grpc_server.UrlTest(client, in.Url, in.Timeout, grpc_server.UrlTestStandard_RTT)

	case gen.TestMode_TcpPing:
		out.Ms, err = grpc_server.TcpPing(in.Address, in.Timeout)

	case gen.TestMode_FullTest:
		// FullTest 现在通过 ProxyCore 接口直接交互
		return grpc_server.DoFullTest(ctx, in, s)

	case gen.TestMode_CheckProxy:
		if i == nil {
			out.Error = "no instance available"
			return
		}
		client := CreateHttpClientForBox(i)
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

		log.Info("IP Info: ", info.Query, location)
		out.FullReport = info.Query + location
	}

	return
}

func (s *server) QueryStats(ctx context.Context, in *gen.QueryStatsReq) (*gen.QueryStatsResp, error) {
	instance := instanceManager.GetInstance()
	if instance == nil {
		return &gen.QueryStatsResp{Traffic: 0}, nil
	}

	v2rayServer := service.FromContext[adapter.V2RayServer](instance.Context())
	if v2rayServer == nil || v2rayServer.StatsService() == nil {
		return &gen.QueryStatsResp{Traffic: 0}, nil
	}
	statsService, ok := v2rayServer.StatsService().(*v2rayapi.StatsService)
	if !ok || statsService == nil {
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
	traffic, err := statsService.GetNekoStats(ctx, name, true)
	if err != nil {
		// missing counter means no traffic yet
		return &gen.QueryStatsResp{Traffic: 0}, nil
	}
	return &gen.QueryStatsResp{Traffic: traffic}, nil
}

func (s *server) ListConnections(ctx context.Context, in *gen.EmptyReq) (*gen.ListConnectionsResp, error) {
	instance := instanceManager.GetInstance()
	if instance == nil {
		payload, _ := json.Marshal([]map[string]any{})
		return &gen.ListConnectionsResp{NekorayConnectionsJson: string(payload)}, nil
	}

	clashServer := service.FromContext[adapter.ClashServer](instance.Context())
	if clashServer == nil {
		return nil, errors.New("no clash server found")
	}
	clash, ok := clashServer.(*clashapi.Server)
	if !ok || clash == nil {
		return nil, errors.New("invalid clash server type")
	}

	connections := clash.TrafficManager().Connections()
	items := make([]map[string]any, 0, len(connections))
	for _, c := range connections {
		if c == nil {
			continue
		}
		item := map[string]any{
			"ID":    c.ID.String(),
			"Dest":  c.Metadata.Destination.String(),
			"RDest": "",
			"Uid":   0,
			"Start": c.CreatedAt.Unix(),
			"End":   0,
			"Tag":   c.Outbound,
		}
		if c.Metadata.Domain != "" {
			item["RDest"] = c.Metadata.Domain
		} else if c.Metadata.Destination.IsFqdn() {
			item["RDest"] = c.Metadata.Destination.Fqdn
		}
		if c.Metadata.ProcessInfo != nil {
			item["Uid"] = c.Metadata.ProcessInfo.UserId
			if c.Metadata.ProcessInfo.ProcessPath != "" {
				item["Process"] = filepath.Base(c.Metadata.ProcessInfo.ProcessPath)
			}
		}
		if !c.ClosedAt.IsZero() {
			item["End"] = c.ClosedAt.Unix()
		}
		items = append(items, item)
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	return &gen.ListConnectionsResp{NekorayConnectionsJson: string(payload)}, nil
}

func (s *server) getOrCreateInstance(config *gen.LoadConfigReq) (*box.Box, func(), error) {
	if config != nil {
		i, cancel, err := boxapi.Create([]byte(config.CoreConfig), grpc_server.CoreLogger)
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
