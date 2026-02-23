# FluxRoute Python SDK (Preview)

`fluxroute-sdk` provides a lightweight control-plane and runtime client for FluxRoute.

## Install (local preview)

```bash
cd sdk/python
python -m pip install -e .
```

## Quickstart

```python
from fluxroute_sdk import FluxRouteClient

client = FluxRouteClient(api_key="demo-key")
client.create_tenant("tenant-a")
client.run("demo/manifests/tenant-a.yaml")
client.add_usage("tenant-a", 50)
print(client.get_invoice("tenant-a"))
```

## Surface

- Router API: `run`, `validate`, `replay`
- Control plane: `create_tenant`, `list_tenants`, `add_usage`, `get_usage`
- Billing: `get_invoice`, `get_invoice_csv`, `get_billing_summary`

## Example app

```bash
python examples/run_and_invoice.py
```
