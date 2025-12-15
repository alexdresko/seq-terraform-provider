# Terraform Provider for Seq

This repository contains a Terraform provider for managing resources in **Seq** using the Seq HTTP API.

Primary focus: **Seq API Keys** (`/api/apikeys`).

Seq API documentation:
- https://datalust.co/docs/using-the-http-api
- https://datalust.co/docs/server-http-api

## Development

### Requirements

- Go 1.22+
- Terraform 1.5+

### Dev container

This repo includes a devcontainer that starts Seq in Docker alongside the development environment.

- Dev container docs: [.devcontainer/README.md](.devcontainer/README.md)
- Seq UI/API (host): `http://localhost:5342`
- Seq URL from inside the devcontainer: `http://seq:80`

### Build

```powershell
go test ./...
go build -o bin/terraform-provider-seq.exe .
```

On Linux/macOS:

```bash
go test ./...
go build -o bin/terraform-provider-seq .
```

### VS Code tasks

Open the Command Palette → **Tasks: Run Task**:
- `go: test`
- `go: fmt`
- `provider: build`
- `docs: generate`

## Provider configuration

```hcl
provider "seq" {
  server_url = "http://localhost:5342"
  api_key    = var.seq_api_key
}
```

Environment variables:
- `SEQ_SERVER_URL`
- `SEQ_API_KEY`
- `SEQ_INSECURE_SKIP_VERIFY`
- `SEQ_TIMEOUT_SECONDS`

## Resources

- `seq_api_key` - manages Seq API keys.

## Data sources

- `seq_health` - reads `/health`.

## Notes

- Seq may only return an API key token on creation. The provider stores the token in state as a **sensitive** attribute and preserves it when Seq does not return it on subsequent reads.

## Publishing to the Terraform Provider Registry

Terraform Registry publishing is automated via GoReleaser + GitHub Actions.

### One-time setup

- GitHub repo requirements (Terraform Registry detection):
  - Repository name must be `terraform-provider-seq` (lowercase).
  - Repository must be public.
  - Add the GitHub topic `terraform-provider`.
- Terraform Registry: sign in with GitHub and add a GPG *public* key under User Settings → Signing Keys.
  - The Registry requires signed releases and does **not** accept default ECC keys; use RSA/DSA.
- GitHub repo secrets (Settings → Secrets and variables → Actions):
  - `GPG_PRIVATE_KEY`: ASCII-armored private key export (e.g. `gpg --armor --export-secret-keys <KEYID>`)
  - `GPG_FINGERPRINT`: the key fingerprint GoReleaser should use for signing

### Releasing

- Create a new semver tag and push it:
  - `./scripts/release-tag.sh 0.1.0`
  - `git push origin v0.1.0`
- GitHub Actions workflow [`.github/workflows/release.yml`](.github/workflows/release.yml) builds multi-platform binaries, generates:
  - `terraform-provider-seq_<VERSION>_<OS>_<ARCH>.zip`
  - `terraform-provider-seq_<VERSION>_manifest.json`
  - `terraform-provider-seq_<VERSION>_SHA256SUMS` and `..._SHA256SUMS.sig`

### Publishing in the Registry UI

- In the Terraform Registry, go to Publish → Provider and select the GitHub repository.
- Once published, future GitHub Releases trigger Registry ingestion via webhook.
