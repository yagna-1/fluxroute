from __future__ import annotations

from datetime import datetime, timezone

from fluxroute_sdk import FluxRouteClient


client = FluxRouteClient(api_key="demo-key")

client.create_tenant("tenant-a")
client.run("demo/manifests/tenant-a.yaml")
client.add_usage("tenant-a", 42)

month = datetime.now(timezone.utc).strftime("%Y-%m")
summary = client.get_billing_summary(month)
invoice = client.get_invoice("tenant-a")

print("summary:", summary)
print("invoice:", invoice)
