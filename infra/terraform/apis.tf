locals {
  required_apis = [
    # The documents bucket in storage.tf.
    "storage.googleapis.com",

    # The runtime service account in service_account.tf.
    "iam.googleapis.com",

    # Every google_*_iam_member resource goes through Resource Manager, so the
    # bucket grant in storage.tf cannot be applied without it.
    "cloudresourcemanager.googleapis.com",
  ]
}

# disable_on_destroy = false: these APIs may already be in use by other things
# in the project. Tearing this root down must not switch them off underneath
# resources this state does not own.
resource "google_project_service" "apis" {
  for_each           = toset(local.required_apis)
  service            = each.value
  disable_on_destroy = false
}
