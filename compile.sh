#!/bin/bash

#================================================================
#   
#   文件名称：compile.sh
#   描    述：Monolithic 统一构建脚本
#
#================================================================
function main() {
	export DEPLOYMENT=$(pwd)/build
	export GOOS=linux
	export GOARCH=amd64
	export TAGS="with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_acme,with_clash_api,with_v2ray_api,with_tailscale,with_ccm,with_ocm,badlinkname,tfogo_checklinkname0"

	pushd sing-box
	export GOHOSTOS="$(go env GOHOSTOS)"
	export GOHOSTARCH="$(go env GOHOSTARCH)"
	export VERSION="$(CGO_ENABLED=0 GOOS=$GOHOSTOS GOARCH=$GOHOSTARCH go run ./cmd/internal/read_tag)"
	popd

	[ "$GOOS" == "windows" ] && [ "$GOARCH" == "amd64" ] && DEST=$DEPLOYMENT/windows64 || true
	[ "$GOOS" == "windows" ] && [ "$GOARCH" == "arm64" ] && DEST=$DEPLOYMENT/windows-arm64 || true
	[ "$GOOS" == "linux" ] && [ "$GOARCH" == "amd64" ] && DEST=$DEPLOYMENT/linux64 || true
	[ "$GOOS" == "linux" ] && [ "$GOARCH" == "arm64" ] && DEST=$DEPLOYMENT/linux-arm64 || true
	if [ -z $DEST ]; then
		echo "Please set GOOS GOARCH"
		exit 1
	fi

	mkdir -p $DEST

	# 核心优化：
	# 1. 直接在根目录通过 package 路径编译入口 ./cmd/nekobox_core
	# 2. 传入 -checklinkname=0 保证 Go 1.25 下跨 package 的符号链接合法化
	go build -v -o $DEST -trimpath -ldflags "-w -s -checklinkname=0 -X github.com/sagernet/sing-box/constant.Version=$VERSION" -tags "$TAGS" ./cmd/nekobox_core
}

main $@
