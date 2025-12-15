provider "seq" {
  # From inside the devcontainer, Seq is reachable at http://seq:80
  # From the host machine, you likely want http://localhost:5342
  server_url = var.seq_server_url

  # NOTE: Hard-coded for experimentation per request.
  # Do not commit real secrets to VCS in production.
  api_key = "HNOY5GNGtEyBNBgDzwrs"
}

variable "seq_server_url" {
  description = "Base URL for the Seq server"
  type        = string
  default     = "http://seq:80"
}
