#!/bin/bash

set -e

mkdir -p ${wgadmin_config_path}
WGADMIN_CONFIG_FILE=${wgadmin_config_path}/${wgadmin_config_file}
cat - > $WGADMIN_CONFIG_FILE <<EOF
${wgadmin_config}
EOF
chmod 0600 $WGADMIN_CONFIG_FILE

add-apt-repository -y ppa:wireguard/wireguard
apt-get update -y
apt-get install -y wireguard=${wireguard_ubuntu_version} linux-headers-$(uname -r)

TAR_FILE="/tmp/wgadmin_$(uname -s)_$(uname -m).tar.gz"
curl -o "$TAR_FILE" -fSL "${wgadmin_releases_url}/${wgadmin_version}/wgadmin_$(uname -s)_$(uname -m).tar.gz"; \
	echo "${wgadmin_version_checksum}  $TAR_FILE" | shasum -c -;

tar -xf "$TAR_FILE" -C /usr/bin/ wgadmin && rm -f $TAR_FILE

cat - >/etc/default/wgadmin <<EOF
WGADMIN_CONFIG_PATH=$WGADMIN_CONFIG_FILE
EOF

wgadmin install-daemons --config-file $WGADMIN_CONFIG_FILE
