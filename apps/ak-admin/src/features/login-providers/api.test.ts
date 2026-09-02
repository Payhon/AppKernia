import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

vi.mock("../auth/store", () => ({
  authSession: {
    adminRequest: (path: string, init?: RequestInit) => fetch(`http://login-provider.test/admin-api/v1${path}`, init),
  },
}));

import {
  createLoginProviderConfig,
  getLoginProviderConfig,
  listAppLoginProviderBindings,
  listLoginProviderConfigs,
  putAppLoginProviderBindings,
  rotateLoginProviderSecret,
} from "./api";
import { loginProviderCodes } from "./model";

const config = (extra: Record<string, unknown> = {}) => ({
  id: "018f08d0-3b00-7000-8000-000000000001",
  name: "GitHub production",
  description: "Production identity",
  provider_code: "github",
  external_client_id: "github-client-id",
  config_schema_version: 1,
  public_config: { app_return_uri: "https://app.example.com/oauth" },
  secret_field_names: ["client_secret"],
  has_secret: true,
  status: "active",
  last_preflight_at: "2026-09-01T00:00:00Z",
  last_preflight_status: "ready",
  last_preflight_issues: [],
  binding_count: 1,
  lock_version: 4,
  created_at: "2026-09-01T00:00:00Z",
  updated_at: "2026-09-01T00:00:00Z",
  callback_uri: "https://api.example.com/oauth/github/callback",
  ...extra,
});

const unboundItems = loginProviderCodes.map((providerCode, index) => ({
  id: null,
  app_id: "018f08d0-3b00-7000-8000-000000000002",
  provider_code: providerCode,
  login_provider_config_id: null,
  config_name: null,
  config_status: null,
  preflight_status: null,
  enabled: false,
  sort_order: (index + 1) * 10,
  lock_version: 0,
  updated_at: null,
}));

const server = setupServer();
beforeAll(() => { server.listen({ onUnhandledRequest: "error" }); });
afterEach(() => { server.resetHandlers(); });
afterAll(() => { server.close(); });

describe("login provider API contracts", () => {
  it("encodes bounded list filters and validates the page response", async () => {
    server.use(http.get("http://login-provider.test/admin-api/v1/login-provider-configs", ({ request }) => {
      const query = new URL(request.url).searchParams;
      expect(query.get("q")).toBe("GitHub & 中文");
      expect(query.get("provider_code")).toBe("github");
      expect(query.get("status")).toBe("active");
      expect(query.get("page")).toBe("2");
      return HttpResponse.json({ data: { items: [config()], page: 2, page_size: 20, total: 1 } });
    }));
    await expect(listLoginProviderConfigs({ q: "GitHub & 中文", provider_code: "github", status: "active", page: 2, page_size: 20 })).resolves.toMatchObject({ total: 1 });
  });

  it("omits lock_version from create and never exposes an echoed Secret", async () => {
    server.use(http.post("http://login-provider.test/admin-api/v1/login-provider-configs", async ({ request }) => {
      const body = await request.json() as Record<string, unknown>;
      expect(body["lock_version"]).toBeUndefined();
      expect(body["client_secret"]).toBeUndefined();
      return HttpResponse.json({ data: config({ client_secret: "must-not-reach-ui" }) });
    }));
    const result = await createLoginProviderConfig({
      name: "GitHub production",
      description: "Production identity",
      provider_code: "github",
      external_client_id: "github-client-id",
      config_schema_version: 1,
      public_config: { app_return_uri: "https://app.example.com/oauth" },
      lock_version: 99,
    });
    expect("client_secret" in result).toBe(false);
  });

  it("refreshes one config without requiring an optional credential fingerprint", async () => {
    server.use(http.get("http://login-provider.test/admin-api/v1/login-provider-configs/:id", ({ params }) => {
      expect(params["id"]).toBe("018f08d0-3b00-7000-8000-000000000001");
      return HttpResponse.json({ data: config({ credential_fingerprint: undefined, lock_version: 5 }) });
    }));
    await expect(getLoginProviderConfig("018f08d0-3b00-7000-8000-000000000001")).resolves.toMatchObject({ lock_version: 5 });
  });

  it("sends plaintext credentials only to the write-only rotation endpoint", async () => {
    server.use(http.post("http://login-provider.test/admin-api/v1/login-provider-configs/:id/rotate-secret", async ({ request }) => {
      expect(await request.json()).toEqual({ values: { client_secret: "one-time-value" }, lock_version: 4 });
      return HttpResponse.json({ data: config({ lock_version: 5 }) });
    }));
    await expect(rotateLoginProviderSecret("018f08d0-3b00-7000-8000-000000000001", { values: { client_secret: "one-time-value" }, lock_version: 4 })).resolves.toMatchObject({ lock_version: 5 });
  });

  it("parses all four null-id unbound rows and sends one atomic bulk PUT", async () => {
    server.use(
      http.get("http://login-provider.test/admin-api/v1/apps/:appId/login-provider-bindings", () => HttpResponse.json({ data: { items: unboundItems } })),
      http.put("http://login-provider.test/admin-api/v1/apps/:appId/login-provider-bindings", async ({ request }) => {
        const body = await request.json() as { bindings: unknown[] };
        expect(body.bindings).toHaveLength(4);
        return HttpResponse.json({ data: { items: unboundItems } });
      }),
    );
    await expect(listAppLoginProviderBindings("018f08d0-3b00-7000-8000-000000000002")).resolves.toHaveLength(4);
    await expect(putAppLoginProviderBindings("018f08d0-3b00-7000-8000-000000000002", { bindings: unboundItems.map((item) => ({ provider_code: item.provider_code, login_provider_config_id: null, enabled: false, sort_order: item.sort_order, lock_version: item.lock_version })) })).resolves.toHaveLength(4);
  });

  it("surfaces a bulk optimistic conflict without replaying the write", async () => {
    let calls = 0;
    server.use(http.put("http://login-provider.test/admin-api/v1/apps/:appId/login-provider-bindings", () => {
      calls += 1;
      return HttpResponse.json({ error: { code: "COMMON.CONFLICT", message: "Stale binding version" } }, { status: 409 });
    }));
    const input = { bindings: unboundItems.map((item) => ({ provider_code: item.provider_code, login_provider_config_id: null, enabled: false, sort_order: item.sort_order, lock_version: item.lock_version })) };
    await expect(putAppLoginProviderBindings("018f08d0-3b00-7000-8000-000000000002", input)).rejects.toMatchObject({ status: 409 });
    expect(calls).toBe(1);
  });
});
