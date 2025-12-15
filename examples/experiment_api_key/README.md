# Experiment: create a Seq API key

This example creates a single `seq_api_key` resource.

## Security note

This example **hard-codes** a Seq API key for quick experimentation:

- `HNOY5GNGtEyBNBgDzwrs`

Do not commit real credentials to version control in production. Prefer `SEQ_API_KEY` or a Terraform variable.

## Prereqs

- Terraform 1.5+
- A running Seq instance
- The provider binary available to Terraform (dev override)

## Run (devcontainer)

1) Build the provider:

From the repo root:

- `go build -o bin/terraform-provider-seq .`

2) Use the provided Terraform CLI config file (`terraform.tfrc`) to load the provider from the local `bin/` directory:

- `export TF_CLI_CONFIG_FILE="$PWD/terraform.tfrc"`

3) Initialize and apply:

- `terraform init`
- `terraform apply`

## Notes

- Default `seq_server_url` is `http://seq:80` (works inside the devcontainer).
- Override if needed:

  - `terraform apply -var 'seq_server_url=http://localhost:5342'`
