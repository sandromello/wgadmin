provider "google" {
  version     = "~> 2.20"
}

locals {
  network_name = "default"
}

resource "google_project" "wgadmin" {
  org_id              = var.gcp_organization_id
  project_id          = var.gcp_project_id
  name                = var.gcp_project_name
  billing_account     = var.gcp_billing_account
  auto_create_network = false
}

resource "google_project_service" "project_services" {
  count = length(var.gcp_activate_apis)

  project = google_project.wgadmin.project_id
  service = element(var.gcp_activate_apis, count.index)

  disable_on_destroy         = false
  disable_dependent_services = true

  depends_on = ["google_project.wgadmin"]
}

resource "google_compute_network" "default" {
  name                    = local.network_name
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
  project                 = google_project.wgadmin.project_id

  depends_on = [
    "google_project_service.project_services"
  ]
}

resource "google_compute_route" "route" {
  project = google_project.wgadmin.project_id
  network = google_compute_network.default.name

  name                   = "egress-internet"
  description            = "route through IGW to access internet"
  tags                   = compact(split(",","egress-inet"))
  dest_range             = "0.0.0.0/0"
  next_hop_gateway       = "default-internet-gateway"
  priority               = "1000"

  depends_on = [
    "google_compute_network.default"
  ]
}

resource "google_compute_firewall" "wireguard-server" {
  name    = "wireguard-server"
  network = google_compute_network.default.name
  project = google_project.wgadmin.project_id

  allow {
    protocol = "udp"
    ports    = ["51820"]
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_address" "static-ip" {
  name    = "wgadmin-static-ip"
  region  = substr(var.gcp_zone, 0, length(var.gcp_zone)-2)
  project = google_project.wgadmin.project_id
}

resource "google_compute_subnetwork" "subnetwork" {
  name                     = "wgadmin-default"
  description              = "wgadmin default subnetwork"
  ip_cidr_range            = cidrsubnet(var.gcp_cidr_subnet.ip_range, var.gcp_cidr_subnet.bits, var.gcp_cidr_subnet.net_num)
  region                   = substr(var.gcp_zone, 0, length(var.gcp_zone)-2)
  private_ip_google_access = false
  network                  = google_compute_network.default.name
  project                  = google_project.wgadmin.project_id
}

resource "google_compute_instance" "default" {
  name         = "wgadmin"
  machine_type = var.gcp_machine_type
  zone         = var.gcp_zone
  project      = google_project.wgadmin.project_id

  can_ip_forward            = true
  allow_stopping_for_update = true

  tags = ["wgadmin"]

  boot_disk {
    initialize_params {
      image = var.gcp_image_name
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.subnetwork.self_link
    access_config {
      nat_ip       = google_compute_address.static-ip.address
      network_tier = var.gcp_network_tier
    }
  }

  service_account {
    scopes = ["storage-ro"]
  }

  metadata_startup_script   = module.wgadmin.ubuntu_install_script
}
