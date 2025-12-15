resource "seq_api_key" "experiment" {
  title       = "terraform-experiment"
  permissions = ["Read"]
}

output "api_key_id" {
  value = seq_api_key.experiment.id
}

output "api_key_token" {
  value     = seq_api_key.experiment.token
  sensitive = true
}
