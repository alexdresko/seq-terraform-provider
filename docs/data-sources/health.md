---
page_title: "seq_health (Data Source)"
description: |-
  Reads the Seq /health endpoint.
---

# seq_health (Data Source)

Reads `/health` and exposes the returned `status` message.

## Example

```terraform
data "seq_health" "this" {}
```
