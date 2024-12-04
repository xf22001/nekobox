package main

import (
	"fmt"
	"libneko/neko_common"
	"os"
	_ "unsafe"

	"grpc_server"

	boxmain "github.com/sagernet/sing-box/cmd/sing-box"
)

func main() {
	fmt.Println()
	// nekobox_core
	if len(os.Args) > 1 && os.Args[1] == "nekobox" {
		neko_common.RunMode = neko_common.RunMode_NekoBox_Core
		grpc_server.RunCore(setupCore, &server{})
		return
	}

	// sing-box
	boxmain.Main()
}
