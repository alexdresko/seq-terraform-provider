terraform {
  required_version = ">= 1.5.0"

  required_providers {
    seq = {
      source = "example/seq"
      # When using provider development overrides (terraform.tfrc), do not pin a
      # version here or Terraform may try to query the public registry.
    }
  }
}
