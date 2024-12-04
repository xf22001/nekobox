package main

import (
	"fmt"
	"os"
	_ "unsafe"

	"grpc_server"

	boxmain "github.com/sagernet/sing-box/cmd/sing-box"
)

func main() {
	fmt.Println()
	// nekobox_core
	if len(os.Args) > 1 && os.Args[1] == "nekobox" {
		grpc_server.RunCore(setupCore, &server{})
		return
	}

	// sing-box
	boxmain.Main()
}
