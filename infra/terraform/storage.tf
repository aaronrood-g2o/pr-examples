# Object storage for documents. The service added in the next PR reads and
# writes objects here, and the HTTP handler after that serves them.
#
# Google-managed encryption keys, not customer-managed (GCP-0066): Google
# encrypts every object either way, and a Cloud KMS key adds cost and key
# rotation with nothing yet requiring it.

#trivy:ignore:GCP-0066
resource "google_storage_bucket" "documents" {
  name     = "${var.project}-documents"
  location = var.region

  # Uniform access means IAM is the only permission system. Per-object ACLs
  # would be a second one that IAM does not show. Public access prevention is
  # enforced rather than inherited, so no later IAM change can make an object
  # world-readable by accident.
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # An overwrite keeps the previous generation instead of losing it. Turn this
  # on before the first mistake, not after it.
  versioning {
    enabled = true
  }

  # No lifecycle rules yet: nothing is known to expire. Add them with the code
  # that writes objects, once there is something to say about retention.
  #
  # No prevent_destroy yet either, because the bucket starts empty. Add
  # `lifecycle { prevent_destroy = true }` the day it holds data you cannot
  # recreate.

  depends_on = [google_project_service.apis]
}

# objectAdmin rather than objectCreator + objectViewer: overwriting an existing
# object needs storage.objects.delete. With versioning on, an overwrite
# archives the old generation instead of destroying it.
resource "google_storage_bucket_iam_member" "documents_object_admin" {
  bucket = google_storage_bucket.documents.name
  role   = "roles/storage.objectAdmin"
  member = google_service_account.documents.member

  # Bucket IAM is checked on every request. Create the new binding before
  # removing the old one, so a change here does not briefly 403 every read and
  # write from a running service.
  lifecycle {
    create_before_destroy = true
  }
}
