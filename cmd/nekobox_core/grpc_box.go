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
			instance = nil
		}
	}()

	if grpc_server.Debug {
		log.Println("Start:", in.CoreConfig)
	}

	if instance != nil {
		err = errors.New("instance already started")
		return
	}

	instance, instance_cancel, err = boxmain.Create([]byte(in.CoreConfig))

	if instance != nil {
		// Logger
		instance.SetLogWritter(neko_log.LogWriter)
	} else {
		log.Println("err:", err)
	}

	return
}

func (s *server) Stop(ctx context.Context, in *gen.EmptyReq) (out *gen.ErrorResp, _ error) {
	var err error

	defer func() {
		out = &gen.ErrorResp{}
		if err != nil {
			out.Error = err.Error()
		}
	}()

	if instance == nil {
		return
	}

	instance_cancel()
	instance.Close()

	instance = nil

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

	if in.Mode == gen.TestMode_UrlTest {
		var i *box.Box
		var cancel context.CancelFunc
		if in.Config != nil {
			// Test instance
			if grpc_server.Debug {
				log.Println("UrlTest:", in.Config.CoreConfig)
			}
			i, cancel, err = boxmain.Create([]byte(in.Config.CoreConfig))
			if i != nil {
				defer i.Close()
				defer cancel()
			}
			if err != nil {
				return
			}
		} else {
			// Test running instance
			i = instance
			if i == nil {
				return
			}
		}
		// Latency
		out.Ms, err = speedtest.UrlTest(boxapi.CreateProxyHttpClient(i), in.Url, in.Timeout, speedtest.UrlTestStandard_RTT)
	} else if in.Mode == gen.TestMode_TcpPing {
		out.Ms, err = speedtest.TcpPing(in.Address, in.Timeout)
	} else if in.Mode == gen.TestMode_FullTest {
		var i *box.Box
		if in.Config != nil {
			if grpc_server.Debug {
				log.Println("FullTest:", in.Config.CoreConfig)
			}
			i, cancel, err := boxmain.Create([]byte(in.Config.CoreConfig))
			if i != nil {
				defer i.Close()
				defer cancel()
			}
			if err != nil {
				return
			}
		} else {
			// Test running instance
			i = instance
			if i == nil {
				return
			}
		}
		if err != nil {
			return
		}
		return grpc_server.DoFullTest(ctx, in, i)
	}

	return
}

func (s *server) QueryStats(ctx context.Context, in *gen.QueryStatsReq) (out *gen.QueryStatsResp, _ error) {
	out = &gen.QueryStatsResp{}

	if instance != nil {
		for _, vs := range instance.Router().GetTrackers() {
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
	}

	return
}

func (s *server) ListConnections(ctx context.Context, in *gen.EmptyReq) (*gen.ListConnectionsResp, error) {
	out := &gen.ListConnectionsResp{
		// TODO upstream api
	}
	for _, vs := range instance.Router().GetTrackers() {
		if cs, ok := vs.(*clashapi.Server); ok {
			connections := cs.TrafficManager().Connections()
			buf := &bytes.Buffer{}
			buf.Reset()

			if err := json.NewEncoder(buf).Encode(connections); err != nil {
				return out, err
			}
			out = &gen.ListConnectionsResp{
				NekorayConnectionsJson: buf.String(),
			}
		}
	}
	return out, nil
}
