#!/bin/bash

#================================================================
#
#
#   文件名称：compile.sh
#   创 建 者：肖飞
#   创建日期：2024年12月04日 星期三 15时14分18秒
#   修改日期：2026年08月27日 星期四 12时00分00秒
#   描    述：nekoray 分支：基于 Xray-core 引擎构建 nekoray_core
#
#================================================================
function main() {
	export DEPLOYMENT=$(pwd)/build
	export GOOS=linux
	export GOARCH=amd64
	export CGO_ENABLED=0

	[ "$GOOS" == "windows" ] && [ "$GOARCH" == "amd64" ] && DEST=$DEPLOYMENT/windows64 || true
	[ "$GOOS" == "windows" ] && [ "$GOARCH" == "arm64" ] && DEST=$DEPLOYMENT/windows-arm64 || true
	[ "$GOOS" == "linux" ] && [ "$GOARCH" == "amd64" ] && DEST=$DEPLOYMENT/linux64 || true
	[ "$GOOS" == "linux" ] && [ "$GOARCH" == "arm64" ] && DEST=$DEPLOYMENT/linux-arm64 || true
	if [ -z $DEST ]; then
		echo "Please set GOOS GOARCH"
		exit 1
	fi

	mkdir -p $DEST

	CGO_ENABLED=$CGO_ENABLED go build -v -o $DEST/nekoray_core -trimpath -ldflags "-w -s" ./cmd/nekoray_core
}

main $@
