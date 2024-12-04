#!/bin/bash

#================================================================
#   
#   
#   文件名称：compile.sh
#   创 建 者：肖飞
#   创建日期：2024年12月04日 星期三 15时14分18秒
#   修改日期：2024年12月04日 星期三 16时27分25秒
#   描    述：
#
#================================================================
function main() {
	export DEPLOYMENT=$(pwd)/build
	export GOOS=linux
	export GOARCH=amd64
	export TAGS_GO120="with_gvisor,with_dhcp,with_wireguard,with_reality_server,with_clash_api,with_quic,with_utls"
	export TAGS_GO121="with_ech"
	export TAGS="$TAGS_GO118,$TAGS_GO120,$TAGS_GO121"

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

	cd cmd/nekobox_core

	go build -v -o $DEST -trimpath -ldflags "-w -s -X github.com/sagernet/sing-box/constant.Version=$VERSION" -tags "$TAGS"
}

main $@
