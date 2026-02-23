import { FluxRouteClient } from "../dist/esm/index.js";

async function main() {
  const client = new FluxRouteClient({ apiKey: "demo-key" });

  await client.createTenant("tenant-a");
  await client.run("demo/manifests/tenant-a.yaml");
  await client.addUsage("tenant-a", 42);

  const month = new Date().toISOString().slice(0, 7);
  const summary = await client.getBillingSummary(month);
  const invoice = await client.getInvoice("tenant-a");

  console.log("summary", summary);
  console.log("invoice", invoice);
}

void main();
