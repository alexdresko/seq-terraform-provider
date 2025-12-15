---
page_title: "seq_api_key (Resource)"
description: |-
  Manages a Seq API key.
---

# seq_api_key (Resource)

Manages API keys via `/api/apikeys`.

Notes:
- Seq may only return the API key `token` at creation time.
- The provider stores `token` in Terraform state as a sensitive value and preserves it when Seq does not return it on subsequent reads.

## Example

```terraform
resource "seq_api_key" "read" {
  title       = "terraform-read"
  permissions = ["Read"]
}
```
