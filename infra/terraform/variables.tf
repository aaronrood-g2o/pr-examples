variable "project" {
  type        = string
  description = "GCP project ID that owns the documents bucket and runtime service account"
}

variable "region" {
  type        = string
  description = "Region for the bucket. Also the provider's default region."
  default     = "us-central1"
}
