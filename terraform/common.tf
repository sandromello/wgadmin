locals {
  wgadmin_config_path  = "/etc/wireguard"
  wgadmin_config_file  = "wgadmin.yaml"
  wgadmin_releases_url = "https://github.com/sandromello/wgadmin/releases/download"
}

variable "wireguard_server_name" {
  description = "The name of the wireguard server"
}

variable "gcs_bucket_name" {
  description = "The name of the GCS bucket to store the bolt db file"
}

variable "server_sync_time" {
  description = "The period in time which the server will be synced"
  default     = "5m"
}

variable "peer_sync_time" {
  description = "The period in time which the peers will be synced"
  default     = "5m"
}

variable "cipher_key" {
  description = "The cipher key to encrypt the server private key"
}

variable "wgadmin_release" {
  type = object({
    version  = string
    checksum = string
  })
  description = "The version and checksum of the wgadmin command line utility"
}

module "configuration" {
  source = "../../modules/configuration"

  server_name      = var.wireguard_server_name
  bucket_name      = var.gcs_bucket_name

  server_sync_time = var.server_sync_time
  peer_sync_time   = var.peer_sync_time
  cipher_key       = var.cipher_key
}

module "wgadmin" {
  source = "../../modules/wgadmin"

  wgadmin_config_path      = local.wgadmin_config_path
  wgadmin_config_file      = local.wgadmin_config_file
  wgadmin_releases_url     = local.wgadmin_releases_url
  wgadmin_version          = var.wgadmin_release.version
  wgadmin_version_checksum = var.wgadmin_release.checksum
  wgadmin_config           = module.configuration.configuration
}

output "install_script" {
  value     = module.wgadmin.install_script
  sensitive = true
}
