from __future__ import annotations

import json
from dataclasses import asdict
from typing import Any
from urllib import error, parse, request

from .models import Invoice, TenantUsage


class FluxRouteError(RuntimeError):
    def __init__(self, message: str, status_code: int | None = None) -> None:
        super().__init__(message)
        self.status_code = status_code


class FluxRouteClient:
    def __init__(
        self,
        router_base_url: str = "http://localhost:8080",
        controlplane_base_url: str = "http://localhost:8081",
        api_key: str | None = None,
        timeout_seconds: int = 15,
    ) -> None:
        self.router_base_url = router_base_url.rstrip("/")
        self.controlplane_base_url = controlplane_base_url.rstrip("/")
        self.api_key = api_key
        self.timeout_seconds = timeout_seconds

    def validate(self, manifest_path: str) -> dict[str, Any]:
        return self._json_request(
            "POST", self.router_base_url, "/v1/validate", body={"manifest_path": manifest_path}
        )

    def run(self, manifest_path: str) -> dict[str, Any]:
        return self._json_request("POST", self.router_base_url, "/v1/run", body={"manifest_path": manifest_path})

    def replay(self, trace_path: str) -> dict[str, Any]:
        return self._json_request("POST", self.router_base_url, "/v1/replay", body={"trace_path": trace_path})

    def create_tenant(self, tenant_id: str, role: str = "admin") -> dict[str, Any]:
        return self._json_request(
            "POST",
            self.controlplane_base_url,
            "/v1/tenants",
            body={"id": tenant_id},
            extra_headers={"X-Role": role},
        )

    def list_tenants(self, q: str = "", page: int = 1, page_size: int = 50) -> dict[str, Any]:
        query = {"page": page, "page_size": page_size}
        if q:
            query["q"] = q
        return self._json_request("GET", self.controlplane_base_url, "/v1/tenants", query=query)

    def add_usage(self, tenant_id: str, invocations: int, role: str = "admin") -> dict[str, Any]:
        return self._json_request(
            "POST",
            self.controlplane_base_url,
            "/v1/usage",
            body={"tenant_id": tenant_id, "invocations": invocations},
            extra_headers={"X-Role": role},
        )

    def get_usage(self, tenant_id: str | None = None, q: str = "", page: int = 1, page_size: int = 50) -> dict[str, Any]:
        query: dict[str, Any] = {"page": page, "page_size": page_size}
        if tenant_id:
            query = {"tenant_id": tenant_id}
        elif q:
            query["q"] = q
        return self._json_request("GET", self.controlplane_base_url, "/v1/usage", query=query)

    def get_invoice(self, tenant_id: str) -> Invoice:
        data = self._json_request(
            "GET", self.controlplane_base_url, "/v1/billing/invoice", query={"tenant_id": tenant_id}
        )
        return Invoice(
            tenant_id=data.get("tenant_id", ""),
            invocations=int(data.get("invocations", 0)),
            usd_per_thousand=float(data.get("usd_per_thousand", 0.0)),
            amount_usd=float(data.get("amount_usd", 0.0)),
        )

    def get_invoice_csv(self, tenant_id: str) -> str:
        return self._text_request(
            "GET",
            self.controlplane_base_url,
            "/v1/billing/invoice",
            query={"tenant_id": tenant_id, "format": "csv"},
        )

    def get_billing_summary(self, month: str) -> dict[str, Any]:
        return self._json_request("GET", self.controlplane_base_url, "/v1/billing/summary", query={"month": month})

    def usage_items(self, q: str = "", page: int = 1, page_size: int = 50) -> list[TenantUsage]:
        payload = self.get_usage(q=q, page=page, page_size=page_size)
        items = payload.get("items", [])
        return [
            TenantUsage(tenant_id=str(item.get("tenant_id", "")), invocations=int(item.get("invocations", 0)))
            for item in items
        ]

    def _json_request(
        self,
        method: str,
        base_url: str,
        path: str,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | None = None,
        extra_headers: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        raw = self._request(method, base_url, path, query=query, body=body, extra_headers=extra_headers)
        if raw.strip() == "":
            return {}
        return json.loads(raw)

    def _text_request(
        self,
        method: str,
        base_url: str,
        path: str,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | None = None,
        extra_headers: dict[str, str] | None = None,
    ) -> str:
        return self._request(method, base_url, path, query=query, body=body, extra_headers=extra_headers)

    def _request(
        self,
        method: str,
        base_url: str,
        path: str,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | None = None,
        extra_headers: dict[str, str] | None = None,
    ) -> str:
        qs = ""
        if query:
            qs = "?" + parse.urlencode(query)
        url = f"{base_url}{path}{qs}"

        headers = {
            "Accept": "application/json",
            "Content-Type": "application/json",
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        if extra_headers:
            headers.update(extra_headers)

        data = None
        if body is not None:
            data = json.dumps(body).encode("utf-8")

        req = request.Request(url, method=method, data=data, headers=headers)
        try:
            with request.urlopen(req, timeout=self.timeout_seconds) as resp:
                return resp.read().decode("utf-8")
        except error.HTTPError as exc:
            payload = exc.read().decode("utf-8")
            raise FluxRouteError(f"HTTP {exc.code}: {payload}", status_code=exc.code) from exc
        except error.URLError as exc:
            raise FluxRouteError(f"request failed: {exc}") from exc


__all__ = ["FluxRouteClient", "FluxRouteError", "Invoice", "TenantUsage", "asdict"]
