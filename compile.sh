#!/bin/bash

#================================================================
#   
#   
#   文件名称：compile.sh
#   创 建 者：肖飞
#   创建日期：2024年12月04日 星期三 15时14分18秒
#   修改日期：2026年06月20日 星期六 18时08分19秒
#   描    述：
#
#================================================================
function main() {
	export DEPLOYMENT=$(pwd)/build
	export GOOS=linux
	export GOARCH=amd64
	export CGO_ENABLED=0
	export TAGS="with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_acme,with_naive_outbound,with_purego,with_clash_api,with_v2ray_api,with_tailscale,with_ccm,with_ocm,badlinkname,tfogo_checklinkname0"

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

	CGO_ENABLED=$CGO_ENABLED go build -v -o $DEST -trimpath -ldflags "-w -s -checklinkname=0 -X github.com/sagernet/sing-box/constant.Version=$VERSION" -tags "$TAGS" ./cmd/nekobox_core

	# with_purego + CGO_ENABLED=0 时 cronet 动态库不会打进二进制，运行时从可执行文件
	# 所在目录加载，必须与 cronet-go 版本配套，否则 naive 出站会段错误，故随产物一起打包
	case "$GOOS/$GOARCH" in
	linux/amd64) CRONET_MODULE=linux_amd64; CRONET_LIB=libcronet.so ;;
	linux/arm64) CRONET_MODULE=linux_arm64; CRONET_LIB=libcronet.so ;;
	windows/amd64) CRONET_MODULE=windows_amd64; CRONET_LIB=libcronet.dll ;;
	windows/arm64) CRONET_MODULE=windows_arm64; CRONET_LIB=libcronet.dll ;;
	*)
		echo "Unsupported platform for cronet: $GOOS/$GOARCH"
		exit 1
		;;
	esac
	CRONET_DIR="$(go list -m -f '{{.Dir}}' github.com/sagernet/cronet-go/lib/$CRONET_MODULE)"
	cp -f "$CRONET_DIR/$CRONET_LIB" "$DEST/"
}

main $@
