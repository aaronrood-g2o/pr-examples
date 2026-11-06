terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # State lives in GCS, like md-portal and collabdocs-api. The bucket is left
  # out here and supplied at init time so this example repo is not tied to one
  # project's state bucket:
  #
  #   terraform init -backend-config="bucket=<state bucket>"
  #
  # If you point this at a bucket another root already uses, KEEP THE PREFIX
  # UNIQUE. Two roots sharing a prefix read each other's state and plan to
  # destroy everything the other one owns.
  backend "gcs" {
    prefix = "pr-examples"
  }
}

provider "google" {
  project = var.project
  region  = var.region
}
