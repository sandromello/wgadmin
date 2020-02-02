# Wireguard Admin

[![Build Status](https://travis-ci.org/sandromello/wgadmin.svg?branch=master)](https://travis-ci.org/sandromello/wgadmin)

Wireguard Admin allows managing servers and peers easily.

## Terraform Bootstrap

```terraform
module "wgadmin" {

}
```

```bash
TF_VAR_cipher_key=$(wgadmin server new-cipher-key) terraform apply
```

# Configure the WebApp

You'll need to configure a Oauth Client ID in order to run the admin webapp. If you already have a project follow the steps below to get all the necessary credentials to run the webapp.

## Configure the Oauth Consent Screen

1. Go to https://console.cloud.google.com/apis/credentials/consent?createClient=&project=<project_id>
2. Select `External` User Type and click on create
3. On Oauth consent screen, select `Public` Application Type
4. Choose an Application Name
5. Put the domain name which will host the wgadmin webapp

## Add an Oauth Client ID

1. Go to https://console.cloud.google.com/apis/credentials?project=<project_id>
2. Click New Credentials, then select OAuth client ID.
3. Select Web Application and fill the name of the app
4. Add the origin and redirect uri using the same address
5. Save it and copy the client id and the client secret

## Create an service account

```bash
SERVICEACCOUNT=wgadmin
PROJECT_ID=system-vpn-d1da18
gcloud iam service-accounts create $SERVICEACCOUNT \
    --description "Wgadmin Webapp" \
    --display-name "wgadmin webapp" \
    --project $PROJECT_ID
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member serviceAccount:$SERVICEACCOUNT@$PROJECT_ID.iam.gserviceaccount.com \
  --role roles/storage.admin \
  --project $PROJECT_ID
```

> **WARNING:** Make sure to run the server with TLS!

```bash
NAMESPACE=wgadmin
GCS_BUCKET_NAME=
GOOGLE_CLIENT_ID=
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

### References
- https://cloud.google.com/docs/authentication/?hl=en_US&_ga=2.221194456.-1500110320.1555950221
- https://support.google.com/googleapi/answer/6158849?hl=en&ref_topic=7013279
- https://cloud.google.com/iam/docs/creating-managing-service-accounts#iam-service-accounts-create-gcloud
- https://cloud.google.com/iam/docs/granting-roles-to-service-accounts#granting_access_to_a_service_account_for_a_resource
- https://cloud.google.com/iam/docs/creating-managing-service-account-keys
