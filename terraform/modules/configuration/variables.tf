variable "server_name" {
  description = "The name of the wireguard server"
}

variable "bucket_name" {
  description = "The name of the GCS bucket to retrieve the wgadmin database"
}

variable "systemd_path" {
  description = "The configuration path of the systemd"
  default     = "/etc/systemd/system"
}

variable "server_unit_name" {
  description = "The systedm unit name for the server runtime process"
  default     = "wgadmin-server.service"
}

variable "peer_unit_name" {
  description = "The systedm unit name for the peer runtime process"
  default     = "wgadmin-peer.service"
}

variable "server_sync_time" {
  description = "The period in time which the server will be synced"
  default     = "5m"
}

variable "peer_sync_time" {
  description = "The period in time which the peers will be synced"
  default     = "5m"
}

variable "config_path" {
  description = "The wgadmin/wireguard config path"
  default     = "/etc/wireguard"
}

variable "config_file" {
  description = "The name of the wireguard config file"
  default     = "wg0.conf"
}

variable "interface_name" {
  description = "The name of the wireguard interface"
  default     = "wg0"
}

variable "cipher_key" {
  description = "The cipher key to encrypt the server private key"
}
