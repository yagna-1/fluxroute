# Runbook: Tenant Onboarding

## Purpose

Create and verify a tenant before workload execution.

## Steps

1. Create tenant:
```bash
curl -X POST http://localhost:8081/v1/tenants \
  -H 'Content-Type: application/json' \
  -H 'X-Role: admin' \
  -H 'X-API-Key: <api-key>' \
  -d '{"id":"tenant-a"}'
```

2. Verify tenant listing:
```bash
curl 'http://localhost:8081/v1/tenants?page=1&page_size=50' \
  -H 'X-API-Key: <api-key>'
```

3. Validate tenant workload manifest namespace isolation:
- Ensure manifest `router.namespace` is tenant-scoped.
- Validate via `/v1/validate` before `/v1/run`.

## Success criteria

- Tenant appears in `GET /v1/tenants`.
- Tenant namespace is explicit in manifest.
