data "template_file" "configuration" {
  template = file("${path.module}/config.yaml")

  vars = {
    server_name      = var.server_name
    bucket_name      = var.bucket_name
    server_unit_name = var.server_unit_name
    peer_unit_name   = var.peer_unit_name
    server_sync_time = var.server_sync_time
    peer_sync_time   = var.peer_sync_time
    systemd_path     = var.systemd_path
    config_path      = var.config_path
    config_file      = var.config_file
    cipher_key       = var.cipher_key
    interface_name   = var.interface_name
  }
}
