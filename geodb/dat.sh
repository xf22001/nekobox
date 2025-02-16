#!/bin/bash

#================================================================
#   
#   
#   文件名称：dat.sh
#   创 建 者：肖飞
#   创建日期：2022年07月10日 星期日 16时28分13秒
#   修改日期：2025年02月16日 星期日 12时00分12秒
#   描    述：
#
#================================================================
function main() {
	##proxychains curl -Lso geoip.dat "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/download/202206042210/geoip.dat"
	##proxychains curl -Lso geosite.dat "https://github.com/v2fly/domain-list-community/releases/download/20220604062951/dlc.dat"
	#proxychains curl -Lso geoip.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
	#proxychains curl -Lso geosite.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat

	user_agent="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.41 Safari/537.36"
	local file_name
	local file_url
	# file_name="geoip.dat"
	# #file_url="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/download/202206042210/geoip.dat"
	# file_url="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
	# wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	# file_name="geosite.dat"
	# #file_url="https://github.com/v2fly/domain-list-community/releases/download/20220604062951/dlc.dat"
	# file_url="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
	# wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	# file_name="geoip.db"
	# file_url="https://github.com/SagerNet/sing-geoip/releases/latest/download/geoip.db"
	# wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	# file_name="geosite.db"
	# file_url="https://github.com/SagerNet/sing-geosite/releases/latest/download/geosite.db"
	# wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	file_name="geoip-cn.srs"
	file_url="https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs"
	wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	file_name="geoip-hk.srs"
	file_url="https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-hk.srs"
	wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	file_name="geoip-tw.srs"
	file_url="https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-tw.srs"
	wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	file_name="geosite-cn.srs"
	file_url="https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs"
	wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"

	file_name="geosite-category-ads-all.srs"
	file_url="https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs"
	wget -t 1 --timeout=30 "$file_url" -O "$file_name" -U "$user_agent"
}

main $@
