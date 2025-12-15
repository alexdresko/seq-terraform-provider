# Acceptance / integration tests

These tests talk to a real Seq instance (the devcontainer's `datalust/seq` container).

## Run

From inside the devcontainer:

- `export SEQ_SERVER_URL=http://seq:80` (optional; defaults to this)
- `export SEQ_API_KEY='...token...'`
- `go test -tags=integration ./internal/acceptance -v`

## Permissions

API key CRUD requires a Seq API key with sufficient permissions (typically `System`).
If the API key is missing or not authorized, the test will be skipped.
