# Runbook: Billing Reconciliation

## Purpose

Reconcile metered usage and invoice outputs for a billing period.

## Steps

1. Pull usage list:
```bash
curl 'http://localhost:8081/v1/usage?page=1&page_size=200' \
  -H 'X-API-Key: <api-key>'
```

2. Pull monthly summary:
```bash
curl 'http://localhost:8081/v1/billing/summary?month=2026-02' \
  -H 'X-API-Key: <api-key>'
```

3. Export per-tenant invoice (JSON/CSV):
```bash
curl 'http://localhost:8081/v1/billing/invoice?tenant_id=tenant-a' \
  -H 'X-API-Key: <api-key>'

curl 'http://localhost:8081/v1/billing/invoice?tenant_id=tenant-a&format=csv' \
  -H 'X-API-Key: <api-key>'
```

4. Compare summary totals vs invoice totals.

## Success criteria

- Monthly summary totals reconcile with invoice exports for all active tenants.
