# Release Checklist

1. Run CI: tests, lint, benchmarks, builds.
2. Confirm replay determinism test coverage is green.
3. Confirm security-sensitive changes reviewed by two maintainers.
4. Build and scan container images.
5. Tag release and publish changelog.
6. Validate control-plane health endpoints and metrics endpoint.
