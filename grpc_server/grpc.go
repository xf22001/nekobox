package grpc_server

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"nekoray/grpc_server/auth"
	"nekoray/grpc_server/gen"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
)

var Debug bool

type BaseServer struct {
	gen.LibcoreServiceServer
	core ProxyCore
}

func (s *BaseServer) setRuntime(core ProxyCore) {
	s.core = core
}

func (s *BaseServer) Exit(ctx context.Context, in *gen.EmptyReq) (out *gen.EmptyResp, _ error) {
	out = &gen.EmptyResp{}

	// Release core resources (running engine instance) before leaving.
	if s.core != nil {
		_ = s.core.Shutdown()
	}

	// Exit the process asynchronously so the RPC response can be delivered to
	// the caller. Calling GracefulStop() here would deadlock: it blocks until
	// all in-flight RPCs finish, including this Exit call itself.
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
	return
}

func RunCore(setupCore func(), server gen.LibcoreServiceServer) {
	_token := flag.String("token", "", "")
	_port := flag.Int("port", 19810, "")
	_debug := flag.Bool("debug", false, "")
	flag.CommandLine.Parse(os.Args[2:])

	Debug = *_debug

	go func() {
		parent, err := os.FindProcess(os.Getppid())
		if err != nil {
			log.Fatal("find parent:", err)
		}
		if runtime.GOOS == "windows" {
			state, err := parent.Wait()
			log.Fatal("parent exited:", state, err)
		} else {
			for {
				time.Sleep(time.Second * 10)
				err = parent.Signal(syscall.Signal(0))
				if err != nil {
					log.Fatal("parent exited:", err)
				}
			}
		}
	}()

	// Engine setup
	setupCore()

	// GRPC
	lis, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(*_port))
	if err != nil {
		log.Fatal("failed to listen: ", err)
	}

	token := *_token
	if token == "" {
		os.Stderr.WriteString("Please set a token: ")
		s := bufio.NewScanner(os.Stdin)
		if s.Scan() {
			token = strings.TrimSpace(s.Text())
		}
	}
	if token == "" {
		fmt.Println("You must set a token")
		os.Exit(0)
	}
	os.Stderr.WriteString("token is set\n")

	auther := auth.Authenticator{
		Token: token,
	}

	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(auther.Authenticate)),
		grpc.UnaryInterceptor(grpc_auth.UnaryServerInterceptor(auther.Authenticate)),
	)

	if bs, ok := server.(interface{ setRuntime(core ProxyCore) }); ok {
		bs.setRuntime(server.(ProxyCore))
	}

	gen.RegisterLibcoreServiceServer(s, server)

	name := "nekoray_core"

	log.Println(name, " grpc server listening at ", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve: ", err)
	}
}
