# wgadmin (GCP)

This Terraform module will provision a project and a single instance that runs the wgadmin daemons.

## Requirements

- terraform 0.12+
- gcloud
- [wgadmin](https://github.com/sandromello/wgadmin/releases)
- gsutil

## Provisioning

1. Authenticate to GCP
2. Create a bucket and a wireguard server
3. Create a `terraform.tfvars` - copy the example below
4. Apply your configuration

```bash
gcloud auth application-default login
terraform init .
TF_VAR_cipher_key=$(wgadmin server new-cipher-key)
terraform apply .
```

> If you wish to use a service account with limited permission to provision this terraform module, you'll need a seed service account. take a look at the [google project factory terraform module](https://github.com/terraform-google-modules/terraform-google-project-factory) for more information.

## terraform.tfvars

```terraform
# the gcp billing account id
gcp_billing_account = ""
# the gcp organization ID
gcp_organization_id = ""
# The project name
gcp_project_name    = "wgadmin"
# The project id
gcp_project_id      = "wgadmin-3a3574"
# The type of machine to create on GCP
gcp_machine_type    = "f1-micro"
# The zone to create the instance
gcp_zone            = "us-east1-b"
# The image of the instance, a non debian image will not work
gcp_image_name      = "ubuntu-os-cloud/ubuntu-1804-lts"
# The network tier of the instance
gcp_network_tier    = "PREMIUM"
# The bucket name to get the bolt database
gcs_bucket_name     = "wgadmin"
# It will create a bucket in the project if the value is true
gcs_create_bucket   = true
# The CIDR range of the subnetwork which the vms will be residing in
gcp_cidr_subnet     = {
  ip_range = "192.168.179.0/24"
  bits     = 5 // 29
  net_num  = 2
}
# The firewall rules that will be created when provisioning the network
gcp_firewall_rules  = [
  {
    protocol = "udp",
    ports    = ["51820"],
  },
  {
    protocol = "tcp",
    ports    = ["22"],
  },
]

# The version of the wireguard
# https://launchpad.net/~wireguard/+archive/ubuntu/wireguard
wireguard_ubuntu_version = "0.0.20191219-wg1~bionic"
# The name of the wireguard server, created with: wgadmin server init (...)
wireguard_server_name    = "myserver"
# The duration in time to sync servers
server_sync_time         = "2m"
# The duration in time to sync peers
peer_sync_time           = "2m"
# The version and checksum of the wgadmin utility, will be download when the vm is provisioned
# https://github.com/sandromello/wgadmin/releases
wgadmin_release          = {
  version  = "v0.0.5-alpha"
  checksum = "d832317c6b2cc5a72291fbd7b0a7bc8167a343edcf11dba3cf2fd4f1ba2e5f26"
}
```

## Customizing

You could leverage the modules directories to use your own terraform assets:

```terraform
module "configuration" {
  source = "github.com/sandromello/wgadmin//terraform/modules/configuration?ref=v0.0.5-alpha"

  server_name      = var.wireguard_server_name
  bucket_name      = var.gcs_bucket_name

  server_sync_time = var.server_sync_time
  peer_sync_time   = var.peer_sync_time
  cipher_key       = var.cipher_key
}

module "wgadmin" {
  source = "github.com/sandromello/wgadmin//terraform/modules/wgadmin?ref=v0.0.5-alpha"

  wgadmin_config_path      = var.wgadmin_config_path
  wgadmin_config_file      = var.wgadmin_config_file
  wgadmin_releases_url     = var.wgadmin_releases_url
  wgadmin_version          = var.wgadmin_release.version
  wgadmin_version_checksum = var.wgadmin_release.checksum
  wgadmin_config           = module.configuration.configuration
  wireguard_ubuntu_version = var.wireguard_ubuntu_version
}

resource "google_compute_instance" "default" {
  (...)

  metadata_startup_script   = module.wgadmin.ubuntu_install_script
}
```

You could install the wgadmin daemons in another distribution, in order to do that you'll need to create a custom installation script for your dist and the system must be compatible with `systemd`, take a look at [ubuntu-install.sh script](../../modules/wgadmin/resources/ubuntu-install.sh) for an example.
