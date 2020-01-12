data "template_file" "install" {
  template = file("${path.module}/resources/install.sh")

  vars = {
    wgadmin_config_path      = var.wgadmin_config_path
    wgadmin_config_file      = var.wgadmin_config_file
    wgadmin_releases_url     = var.wgadmin_releases_url
    wgadmin_version          = var.wgadmin_version
    wgadmin_version_checksum = var.wgadmin_version_checksum
    wgadmin_config           = var.wgadmin_config
    wireguard_ubuntu_version = var.wireguard_ubuntu_version
  }
}
