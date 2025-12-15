terraform {
  required_providers {
    seq = {
      source  = "alexdresko/seq"
      version = ">= 0.0.0"
    }
  }
}

provider "seq" {
  server_url = "http://localhost:5342"
  api_key    = "REDACTED"
}

data "seq_health" "this" {}
