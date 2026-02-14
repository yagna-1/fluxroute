# Security Policy

## Reporting vulnerabilities

Report security issues privately to the maintainers before public disclosure.
Do not open public issues for active vulnerabilities.

## Secure defaults

- Enable mTLS for metrics and control-plane endpoints in production.
- Restrict control-plane admin actions to `admin` role only.
- Use namespace isolation and coordination locks for multi-tenant deployments.
- Persist audit logs and export periodically for compliance workflows.

## Hardening checklist

- Set `REQUEST_ROLE` via trusted auth gateway, not user input.
- Configure `AUDIT_LOG_PATH` to durable storage.
- Use `COORDINATION_MODE=redis` for multi-instance deployments.
- Rotate TLS certs and provider API keys regularly.
