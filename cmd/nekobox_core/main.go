package main

import (
	"fmt"
	"os"
	_ "unsafe"

	"nekobox/grpc_server"

	boxmain "github.com/sagernet/sing-box/cmd/sing-box"
)

func main() {
	// 恢复位置判断：只有当第一个参数明确是 nekobox 时，才进入专有逻辑
	// 这样可以彻底避免误判文件名或路径中的关键字
	if len(os.Args) > 1 && os.Args[1] == "nekobox" {
		fmt.Println("Starting Nekobox Core Service...")
		grpc_server.RunCore(setupCore, &server{})
		return
	}

	// 其他情况（包括 version, help, 以及所有原生 sing-box 命令）
	// 完全交由 sing-box 的 Cobra 框架处理
	boxmain.Main()
}
