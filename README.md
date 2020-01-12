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
NAMESPACE=wgadmin
PROJECT_ID=wgadmin-$(openssl rand -hex 3)
GCS_BUCKET_NAME=$PROJECT_ID
GOOGLE_CLIENT_ID=
# Create a GCS bucket
# gsutil mb -p $PROJECT_ID -c nearline gs://$GCS_BUCKET_NAME
kubectl create ns $NAMESPACE
kubectl create secret -n $NAMESPACE generic tls-ssl-wgadm \
    --from-file=tls-cert=path/to/tls-cert.pem \
    --from-file=tls-cert-key=path/to/tle-cert-key.pem
# Create a service account with Storage Admin role
GOOGLE_APPLICATION_CREDENTIALS=path/to/service-account.json
kubectl create secret -n $NAMESPACE generic google-credentials \
    --from-file=serviceaccount=$GOOGLE_APPLICATION_CREDENTIALS
cat - >webapp-config.yml <<EOF
httpPort: '8000'
allowedDomains:
- acme.tld
- gmail.com
pageConfig:
  faviconURL: null
  googleClientID: $GOOGLE_CLIENT_ID
  googleRedirectURI: https://acme.tld
tlsKeyFile: /etc/ssl/custom-certs/tls-cert-key.pem
tlsCertFile: /etc/ssl/custom-certs/tls-cert.pem
googleApplicationCredentials: /var/run/secrets/google/serviceaccount
gcsBucketName: $GCS_BUCKET_NAME
EOF
kubectl create secret -n $NAMESPACE generic webapp-config --from-file=config.yaml=./webapp-config.yml
kubectl apply -f deploy/webapp/all.yml
```


References:
- https://cloud.google.com/docs/authentication/?hl=en_US&_ga=2.221194456.-1500110320.1555950221

https://console.cloud.google.com/apis/credentials?project=system-vpn-a

> **Note:** This is a working project, come back later.
