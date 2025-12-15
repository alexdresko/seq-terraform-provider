resource "seq_api_key" "example" {
  title       = "terraform-example"
  permissions = ["Read"]
}

# Example with input settings for filtering and tagging events
resource "seq_api_key" "app_ingest" {
  title         = "my-application"
  permissions   = ["Ingest"]
  minimum_level = "Warning"
  filter        = "@Level = 'Error' or @Level = 'Fatal'"

  applied_properties = {
    Application = "MyApp"
    Environment = "Production"
  }
}
