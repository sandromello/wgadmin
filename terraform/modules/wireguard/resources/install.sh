#!/bin/bash

set -e

WGADMIN_CONFIG_PATH=$1
# mv /vagrant/wgadmin /usr/bin/wgadmin

if [ -z $WGADMIN_CONFIG_PATH ]; then
    echo "Missing argument! Run as $0 <config-file>"
    exit 2
fi

add-apt-repository -y ppa:wireguard/wireguard
apt-get update -y
apt-get install -y wireguard linux-headers-$(uname -r)
# TODO: add process to download wgadmin command line utility

# if [ -f /vagrant/gcs-credentials.json ]; then
#     # read-only credentials
#     export GOOGLE_APPLICATION_CREDENTIALS=/vagrant/gcs-credentials.json
#     GCS_BUCKET_NAME=$GCS_BUCKET_NAME wgadmin server init --endpoint $WGADMIN_SERVER:51820 $WGADMIN_SERVER --override
# fi

cat - >/etc/default/wgadmin <<EOF
WGADMIN_CONFIG_PATH=
GOOGLE_APPLICATION_CREDENTIALS=$GOOGLE_APPLICATION_CREDENTIALS
EOF

wgadmin install-daemons --config-file $WGADMIN_CONFIG_PATH
