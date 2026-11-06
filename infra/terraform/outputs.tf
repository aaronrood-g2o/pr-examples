# What the document service needs to be configured with.

output "documents_bucket" {
  description = "Name of the documents bucket"
  value       = google_storage_bucket.documents.name
}

output "documents_service_account" {
  description = "Email of the service account that reads and writes the documents bucket"
  value       = google_service_account.documents.email
}
