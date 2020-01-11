output "install_script" {
  value = data.template_file.install.rendered
}
