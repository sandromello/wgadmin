variable "gcp_project_id" {
  description = "The GCP project id"
}

variable "gcp_project_name" {
  description = "The GCP project name"
}

variable "gcp_machine_type" {
  description = "The GCP instance type"
  default     = "f1-micro"
}

variable "gcp_zone" {
  description = "The name of the gcp zone"
}

variable "gcp_cidr_subnet" {
  type = object({
    ip_range  = string
    bits      = number
    net_num   = number
  })
  description = <<EOF
    The subnetwork CIDR.
    See https://www.terraform.io/docs/configuration-0-11/interpolation.html#cidrsubnet-iprange-newbits-netnum-
    EOF
}

variable "gcp_image_name" {
  description = "The name of the image for the machine"
  default     = "ubuntu-os-cloud/ubuntu-1804-lts"
}

variable "gcp_network_tier" {
  description = "The network tier config"
  default     = "PREMIUM"
}
