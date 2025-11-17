variable "project_id" {
  description = "The ID of the GCP project"
  type        = string
}

variable "region" {
  description = "The GCP region to deploy in"
  type        = string
  default     = "europe-west1"
}

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
  default     = "zero-downtime-gcp-cluster"
}

variable "node_count" {
  description = "Number of nodes in the default node pool"
  type        = number
  default     = 1
}

variable "machine_type" {
  description = "Machine type for nodes"
  type        = string
  default     = "c2-standard-8"
}
