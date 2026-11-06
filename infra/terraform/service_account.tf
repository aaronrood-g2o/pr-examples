# The identity the document service runs as. Its only grant is objectAdmin on
# the documents bucket (storage.tf). It holds no project-level storage role, so
# it cannot reach any other bucket in the project.
resource "google_service_account" "documents" {
  account_id   = "pr-examples-documents"
  display_name = "pr-examples document service"
  description  = "Runtime identity for the pr-examples document service. Reads and writes the documents bucket only."

  depends_on = [google_project_service.apis]
}
