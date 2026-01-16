package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"grpc_server"
	"grpc_server/gen"
	"libneko/boxapi"
	"libneko/neko_log"
	"libneko/speedtest"

	box "github.com/sagernet/sing-box"
	boxmain "github.com/sagernet/sing-box/cmd/sing-box"
	"github.com/sagernet/sing-box/experimental/clashapi"
	"github.com/sagernet/sing-box/experimental/v2rayapi"

	"log"
)

type server struct {
	grpc_server.BaseServer
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
		log.Println("Start:", in.CoreConfig)
	}

	currentInstance := instanceManager.GetInstance()
	if currentInstance != nil {
		return &gen.ErrorResp{Error: "instance already started"}, nil
	}

	newInstance, newCancel, err := boxmain.Create([]byte(in.CoreConfig))
	if err != nil {
		return &gen.ErrorResp{Error: err.Error()}, nil
	}

	if newInstance != nil {
		// Logger
		newInstance.SetLogWritter(neko_log.LogWriter)
		instanceManager.SetInstance(newInstance, newCancel)
	} else {
		log.Println("err:", err)
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
		// Latency
		out.Ms, err = speedtest.UrlTest(boxapi.CreateProxyHttpClient(i), in.Url, in.Timeout, speedtest.UrlTestStandard_RTT)
	case gen.TestMode_TcpPing:
		out.Ms, err = speedtest.TcpPing(in.Address, in.Timeout)
	case gen.TestMode_FullTest:
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
		return grpc_server.DoFullTest(ctx, in, i)
	}

	return
}

// getOrCreateInstance 获取现有实例或创建新实例
func (s *server) getOrCreateInstance(config *gen.LoadConfigReq) (*box.Box, func(), error) {
	if config != nil {
		// 创建临时实例
		if grpc_server.Debug {
			log.Println("Creating temporary instance for test")
		}
		i, cancel, err := boxmain.Create([]byte(config.CoreConfig))
		if err != nil {
			return nil, nil, err
		}
		if i == nil {
			return nil, nil, errors.New("instance creation failed")
		}

		// 返回实例和清理函数
		cleanup := func() {
			cancel()
			i.Close()
		}
		return i, cleanup, nil
	} else {
		// 使用运行中的实例
		instance, exists := instanceManager.GetOrEmpty()
		if !exists {
			return nil, nil, errors.New("no running instance available")
		}
		return instance, nil, nil
	}
}

func (s *server) QueryStats(ctx context.Context, in *gen.QueryStatsReq) (out *gen.QueryStatsResp, _ error) {
	out = &gen.QueryStatsResp{}

	instanceManager.ExecuteWithInstance(func(i *box.Box) error {
		for _, vs := range i.Router().GetTrackers() {
			if ss, ok := vs.(*v2rayapi.StatsService); ok {
				var err error
				//log.Println("tag:", in.Tag, "direct:", in.Direct)
				out.Traffic, err = ss.GetNekoStats(ctx, fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", in.Tag, in.Direct), true)
				//log.Println("traffic:", out.Traffic)
				if err != nil {
					log.Println("GetNekoStats", err.Error())
				}
			}
		}
		return nil
	})

	return
}

func (s *server) ListConnections(ctx context.Context, in *gen.EmptyReq) (*gen.ListConnectionsResp, error) {
	out := &gen.ListConnectionsResp{
		// TODO upstream api
	}

	err := instanceManager.ExecuteWithInstance(func(i *box.Box) error {
		for _, vs := range i.Router().GetTrackers() {
			if cs, ok := vs.(*clashapi.Server); ok {
				connections := cs.TrafficManager().Connections()
				buf := &bytes.Buffer{}
				buf.Reset()

				if err := json.NewEncoder(buf).Encode(connections); err != nil {
					return err
				}
				out = &gen.ListConnectionsResp{
					NekorayConnectionsJson: buf.String(),
				}
			}
		}
		return nil
	})

	if err != nil {
		return out, err
	}

	return out, nil
}
