package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nekoray/grpc_server"

	"github.com/xtls/xray-core/xrayapi"
)

func main() {
	fmt.Println()
	// nekoray_core: gRPC 核心服务模式（供 pynekoray 客户端使用）
	if len(os.Args) > 1 && os.Args[1] == "nekoray" {
		fmt.Println("Starting Nekoray Core Service...")
		grpc_server.RunCore(setupCore, &server{})
		return
	}

	// CLI 模式：运行一个配置文件（服务端部署，兼容 ./nekoray_core run -c config.json）
	runServer(os.Args[1:])
}

func runServer(args []string) {
	// 兼容 ./nekoray_core run -c config.json 与 ./nekoray_core -c config.json
	if len(args) > 0 && args[0] == "run" {
		args = args[1:]
	}
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configFile := fs.String("c", "config.json", "config file")
	fs.Parse(args)

	content, err := os.ReadFile(*configFile)
	if err != nil {
		fmt.Println("read config failed:", err)
		os.Exit(1)
	}

	instance, err := xrayapi.Create(content)
	if err != nil {
		fmt.Println("start instance failed:", err)
		os.Exit(1)
	}
	defer instance.Close()
	fmt.Println("Xray instance started, config:", *configFile)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
