resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.region

  remove_default_node_pool = true
  initial_node_count       = 1

  networking_mode = "VPC_NATIVE"

  ip_allocation_policy {}

  node_config {
    disk_size_gb = 50
  }
}

resource "google_container_node_pool" "primary_nodes" {
  name     = "ollama-node-pool"
  location = var.region
  cluster  = google_container_cluster.primary.name

  node_count = var.node_count

  node_config {
    machine_type = var.machine_type
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]

    # Use faster SSD boot disks
    disk_type    = "pd-standard"
    disk_size_gb = 50

    labels = {
      role = "ollama-node"
    }

    tags = ["ollama-demo"]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}
