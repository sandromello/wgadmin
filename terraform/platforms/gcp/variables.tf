variable "gcp_project_id" {
  description = "The GCP project id"
}

variable "gcp_project_name" {
  description = "The GCP project name"
}

variable "gcp_billing_account" {
  description = "The ID of the billing account to associate this project with"
}

variable "gcp_organization_id" {
  description = "The organization ID"
}

variable "gcp_machine_type" {
  description = "The GCP instance type"
  default     = "f1-micro"
}

variable "gcp_zone" {
  description = "The name of the gcp zone"
}

variable "gcp_activate_apis" {
  type        = list(string)
  description = "A list of apis to activate, execute 'gcloud services list' to see the list of API's"
  default     = [
    "compute.googleapis.com",
  ]
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

variable "gcp_firewall_rules" {
  description = "The firewall ports/protocol to open for the wireguard instances"
  type = set(object({
    protocol = string
    ports    = list(string)
  }))
  default = [{
    protocol = "udp",
    ports    = ["51820"],
  }]
}

variable "gcp_firewall_source_ranges" {
  description = "Which CIDR to apply firewall rules"
  default     = ["0.0.0.0/0"]
}

variable "gcs_create_bucket" {
  description = "Creates a bucket to store the wgadmin database."
  default     = false
}

variable "gcs_bucket_location" {
  description = "The location of the bucket"
  default     = "US"
}

variable "gcs_bucket_storage_class" {
  description = "The Storage Class of the new bucket. Supported values: STANDARD|MULTI_REGIONAL|REGIONAL|NEARLINE|COLDLINE"
  default     = "STANDARD"
}

variable "gcs_bucket_force_destroy" {
  description = "When deleting a bucket, this boolean option will delete all contained objects. If you try to delete a bucket that contains objects, Terraform will fail that run."
  default     = false
}

variable "gcs_bucket_versioning" {
  description = "While set to true, versioning is fully enabled for this bucket"
  default     = false
}
