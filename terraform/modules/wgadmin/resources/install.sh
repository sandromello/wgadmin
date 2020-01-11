#!/bin/bash

set -e

cat - > ${wgadmin_config_path} <<EOF
${wgadmin_config}
EOF
chmod 0600 ${wgadmin_config_path}

add-apt-repository -y ppa:wireguard/wireguard
apt-get update -y
apt-get install -y wireguard linux-headers-$(uname -r)

TAR_FILE="/tmp/wgadmin_$(uname -s)_$(uname -m).tar.gz"
curl -o "$TAR_FILE" -fSL "${wgadmin_releases_url}/${wgadmin_version}/wgadmin_$(uname -s)_$(uname -m).tar.gz"; \
	echo "${wgadmin_version_checksum}  ${TAR_FILE}" | shasum -c -;

tar -xf "$TAR_FILE" -C /usr/local/bin/ wgadmin && rm -f $TAR_FILE

# if [ -f /vagrant/gcs-credentials.json ]; then
#     # read-only credentials
#     export GOOGLE_APPLICATION_CREDENTIALS=/vagrant/gcs-credentials.json
#     GCS_BUCKET_NAME=$GCS_BUCKET_NAME wgadmin server init --endpoint $WGADMIN_SERVER:51820 $WGADMIN_SERVER --override
# fi

cat - >/etc/default/wgadmin <<EOF
WGADMIN_CONFIG_PATH=${wgadmin_config_path}
EOF

wgadmin install-daemons --config-file ${wgadmin_config_path}
