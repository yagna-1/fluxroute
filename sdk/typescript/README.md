# FluxRoute TypeScript SDK (Preview)

`@fluxroute/sdk` provides a typed client for FluxRoute router and control-plane APIs.

## Install

```bash
cd sdk/typescript
npm install
npm run build
```

## Quickstart

```ts
import { FluxRouteClient } from "@fluxroute/sdk";

const client = new FluxRouteClient({ apiKey: "demo-key" });
await client.createTenant("tenant-a");
await client.run("demo/manifests/tenant-a.yaml");
await client.addUsage("tenant-a", 50);
console.log(await client.getInvoice("tenant-a"));
```

## Surface

- Router API: `run`, `validate`, `replay`
- Control plane: `createTenant`, `listTenants`, `addUsage`, `getUsage`
- Billing: `getInvoice`, `getInvoiceCSV`, `getBillingSummary`

## Example app

```bash
npx ts-node examples/runAndInvoice.ts
```
