import type {
  BillingSummary,
  FluxRouteClientOptions,
  Invoice,
  UsageListResponse,
} from "./types.js";

export class FluxRouteError extends Error {
  readonly statusCode?: number;

  constructor(message: string, statusCode?: number) {
    super(message);
    this.name = "FluxRouteError";
    this.statusCode = statusCode;
  }
}

export class FluxRouteClient {
  private readonly routerBaseUrl: string;
  private readonly controlPlaneBaseUrl: string;
  private readonly apiKey?: string;
  private readonly timeoutMs: number;

  constructor(options: FluxRouteClientOptions = {}) {
    this.routerBaseUrl = (options.routerBaseUrl ?? "http://localhost:8080").replace(/\/$/, "");
    this.controlPlaneBaseUrl = (options.controlPlaneBaseUrl ?? "http://localhost:8081").replace(/\/$/, "");
    this.apiKey = options.apiKey;
    this.timeoutMs = options.timeoutMs ?? 15000;
  }

  validate(manifestPath: string): Promise<unknown> {
    return this.requestJSON("POST", this.routerBaseUrl, "/v1/validate", { manifest_path: manifestPath });
  }

  run(manifestPath: string): Promise<unknown> {
    return this.requestJSON("POST", this.routerBaseUrl, "/v1/run", { manifest_path: manifestPath });
  }

  replay(tracePath: string): Promise<unknown> {
    return this.requestJSON("POST", this.routerBaseUrl, "/v1/replay", { trace_path: tracePath });
  }

  createTenant(tenantID: string, role = "admin"): Promise<unknown> {
    return this.requestJSON("POST", this.controlPlaneBaseUrl, "/v1/tenants", { id: tenantID }, role);
  }

  listTenants(page = 1, pageSize = 50, q = ""): Promise<unknown> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    if (q !== "") {
      params.set("q", q);
    }
    return this.requestJSON("GET", this.controlPlaneBaseUrl, `/v1/tenants?${params.toString()}`);
  }

  addUsage(tenantID: string, invocations: number, role = "admin"): Promise<unknown> {
    return this.requestJSON(
      "POST",
      this.controlPlaneBaseUrl,
      "/v1/usage",
      { tenant_id: tenantID, invocations },
      role,
    );
  }

  getUsage(tenantID?: string, page = 1, pageSize = 50, q = ""): Promise<UsageListResponse | unknown> {
    const params = new URLSearchParams();
    if (tenantID && tenantID !== "") {
      params.set("tenant_id", tenantID);
    } else {
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      if (q !== "") {
        params.set("q", q);
      }
    }
    return this.requestJSON("GET", this.controlPlaneBaseUrl, `/v1/usage?${params.toString()}`);
  }

  getInvoice(tenantID: string): Promise<Invoice> {
    return this.requestJSON("GET", this.controlPlaneBaseUrl, `/v1/billing/invoice?tenant_id=${encodeURIComponent(tenantID)}`);
  }

  getInvoiceCSV(tenantID: string): Promise<string> {
    return this.requestText(
      "GET",
      this.controlPlaneBaseUrl,
      `/v1/billing/invoice?tenant_id=${encodeURIComponent(tenantID)}&format=csv`,
    );
  }

  getBillingSummary(month: string): Promise<BillingSummary> {
    return this.requestJSON("GET", this.controlPlaneBaseUrl, `/v1/billing/summary?month=${encodeURIComponent(month)}`);
  }

  private async requestJSON(
    method: string,
    baseUrl: string,
    path: string,
    body?: unknown,
    role?: string,
  ): Promise<any> {
    const response = await this.request(method, baseUrl, path, body, role);
    const text = await response.text();
    if (text.trim() === "") {
      return {};
    }
    try {
      return JSON.parse(text);
    } catch {
      return { raw: text };
    }
  }

  private async requestText(
    method: string,
    baseUrl: string,
    path: string,
    body?: unknown,
    role?: string,
  ): Promise<string> {
    const response = await this.request(method, baseUrl, path, body, role);
    return response.text();
  }

  private async request(
    method: string,
    baseUrl: string,
    path: string,
    body?: unknown,
    role?: string,
  ): Promise<Response> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);

    const headers: Record<string, string> = {
      Accept: "application/json",
      "Content-Type": "application/json",
    };
    if (this.apiKey) {
      headers["X-API-Key"] = this.apiKey;
    }
    if (role) {
      headers["X-Role"] = role;
    }

    try {
      const response = await fetch(`${baseUrl}${path}`, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal,
      });
      if (!response.ok) {
        const payload = await response.text();
        throw new FluxRouteError(`HTTP ${response.status}: ${payload}`, response.status);
      }
      return response;
    } catch (err) {
      if (err instanceof FluxRouteError) {
        throw err;
      }
      throw new FluxRouteError(`request failed: ${String(err)}`);
    } finally {
      clearTimeout(timer);
    }
  }
}
