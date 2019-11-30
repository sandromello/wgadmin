#!/bin/bash

set -e

WGADMIN_SERVER=$1
GCS_BUCKET_NAME=wireguard
WGADMIN_PEER_SYNC=5m
WGADMIN_SERVER_SYNC=5m

# mv /vagrant/wgadmin /usr/bin/wgadmin

if [ -z $WGADMIN_SERVER ]; then
    echo "Missing argument! Run as $0 <wg-server>"
    exit 2
fi

add-apt-repository -y ppa:wireguard/wireguard
apt-get update -y
apt-get install -y wireguard linux-headers-$(uname -r)
# TODO: add process to download wgadmin command line utility

if [ -f /vagrant/gcs-credentials.json ]; then
    # read-only credentials
    export GOOGLE_APPLICATION_CREDENTIALS=/vagrant/gcs-credentials.json
    GCS_BUCKET_NAME=$GCS_BUCKET_NAME wgadmin server init --endpoint $WGADMIN_SERVER:51820 $WGADMIN_SERVER --override
    WGADMIN_PEER_SYNC=30s
fi

cat - >/etc/default/wgadmin <<EOF
WGADMIN_SERVER=$WGADMIN_SERVER
WGADMIN_CONFIG_FILE=/etc/wireguard/wg0.conf
WGADMIN_SERVER_SYNC=$WGADMIN_SERVER_SYNC
WGADMIN_PEER_SYNC=$WGADMIN_PEER_SYNC
GCS_BUCKET_NAME=$GCS_BUCKET_NAME
GOOGLE_APPLICATION_CREDENTIALS=$GOOGLE_APPLICATION_CREDENTIALS
EOF

wgadmin configure systemd-server
wgadmin configure systemd-peer
