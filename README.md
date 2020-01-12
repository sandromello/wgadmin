# Wireguard Admin

[![Build Status](https://travis-ci.org/sandromello/wgadmin.svg?branch=master)](https://travis-ci.org/sandromello/wgadmin)

Wireguard Admin allows managing servers and peers easily.

## Terraform Bootstrap

TODO

## Configure Authentication

You'll need to configure a Oauth Client ID in order to run the admin webapp. If you already have a project follow the steps below to get all the necessary credentials to run the webapp:

1. Go to https://console.cloud.google.com/apis/credentials?project=<YOUR_PROJECT>
2. Create an Oauth Client ID credential
3. Choose "Web Application" and set a name for it
4. Add authorized redirect URI and Origin to your root domain name, example: https://acme.tld

> **WARNING:** Make sure to run the server with TLS!

```bash
kubectl create ns wgadmin
kubectl create secret -n wgadmin generic tls-ssl-wgadm \
    --from-file=tls-cert=path/to/tls-cert.pem \
    --from-file=tls-cert-key=path/to/tls-cert-key.pem
# GOOGLE_APPLICATION_CREDENTIALS=
# kubectl create secret -n wgadmin generic google-credentials \
#     --from-file=serviceaccount=$GOOGLE_APPLICATION_CREDENTIALS \
#     --from-file=GCS_BUCKET_NAME=wgadmin-$(openssl rand -hex 3)

kubectl create secret -n wgadmin generic webapp-config --from-file=config.yaml=./webapp-config.yml
```


References:
- https://cloud.google.com/docs/authentication/?hl=en_US&_ga=2.221194456.-1500110320.1555950221

https://console.cloud.google.com/apis/credentials?project=system-vpn-a

> **Note:** This is a working project, come back later.
