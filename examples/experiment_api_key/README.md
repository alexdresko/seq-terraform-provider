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

### Quick start (scripts)

From this directory:

```bash
./plan.sh      # Build provider + terraform plan
./apply.sh     # Build provider + terraform apply
./destroy.sh   # Terraform destroy (no rebuild)
./clean.sh     # Remove .terraform, state files
```

### Manual steps

1) Build the provider (from repo root):

```bash
go build -o bin/terraform-provider-seq .
```

2) Set CLI config to use local provider:

```bash
export TF_CLI_CONFIG_FILE="$PWD/terraform.tfrc"
```

3) Initialize and apply:

```bash
terraform init
terraform apply
```

## Notes

- Default `seq_server_url` is `http://seq:80` (works inside the devcontainer).
- Override if needed:

  - `terraform apply -var 'seq_server_url=http://localhost:5342'`
