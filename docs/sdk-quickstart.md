# SDK Quickstart

## Python

```bash
cd sdk/python
python -m pip install -e .
python examples/run_and_invoice.py
```

## TypeScript

```bash
cd sdk/typescript
npm install
npm run build
npx ts-node examples/runAndInvoice.ts
```

## API parity

Both SDKs cover:
- Router: run / validate / replay
- Control plane: tenants / usage
- Billing: invoice (JSON + CSV) / monthly summary
