from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class TenantUsage:
    tenant_id: str
    invocations: int


@dataclass(frozen=True)
class Invoice:
    tenant_id: str
    invocations: int
    usd_per_thousand: float
    amount_usd: float
