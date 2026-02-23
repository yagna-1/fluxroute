export interface TenantUsage {
  tenant_id: string;
  invocations: number;
}

export interface UsageListResponse {
  items: TenantUsage[];
  total: number;
  page: number;
  page_size: number;
}

export interface Invoice {
  tenant_id: string;
  invocations: number;
  usd_per_thousand: number;
  amount_usd: number;
}

export interface BillingSummary {
  month: string;
  totals: TenantUsage[];
  grand_total_invocations: number;
}

export interface FluxRouteClientOptions {
  routerBaseUrl?: string;
  controlPlaneBaseUrl?: string;
  apiKey?: string;
  timeoutMs?: number;
}
