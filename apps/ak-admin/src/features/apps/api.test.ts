import { beforeEach, describe, expect, it, vi } from "vitest";

const { adminRequest } = vi.hoisted(() => ({ adminRequest: vi.fn() }));
vi.mock("../auth/store", () => ({ authSession: { adminRequest } }));

import { appAdminApi } from "./api";
import type { AppPageInput } from "./model";

const response = () => new Response(JSON.stringify({ data: {} }), { status: 200, headers: { "Content-Type": "application/json" } });
const requestCalls = (): [string, string | undefined][] => adminRequest.mock.calls.map((call: unknown[]) => [String(call[0]), (call[1] as RequestInit | undefined)?.method]);
const jsonBody = (call: unknown[] | undefined): Record<string, unknown> => {
  const parsed: unknown = JSON.parse((call?.[1] as RequestInit).body as string);
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) throw new Error("expected JSON object");
  return parsed as Record<string, unknown>;
};

describe("App management admin API requests", () => {
  beforeEach(() => { adminRequest.mockReset(); adminRequest.mockImplementation(() => Promise.resolve(response())); });

  it("uses the application list, create, update, and status contracts", async () => {
    await appAdminApi.list({ q: "alpha", page: 2, page_size: 20 });
    await appAdminApi.create({ name: "Alpha", default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp" });
    await appAdminApi.update("app/a", { name: "Alpha", default_locale: "en-US", registration_enabled: false, registration_verification_mode: "none", lock_version: 3 });
    await appAdminApi.setStatus("app/a", "disable", 4);
    expect(requestCalls()).toEqual([
      ["/apps?q=alpha&page=2&page_size=20", undefined], ["/apps", "POST"], ["/apps/app%2Fa", "PATCH"], ["/apps/app%2Fa/disable", "POST"],
    ]);
    expect(jsonBody(adminRequest.mock.calls[1])).toMatchObject({ registration_verification_mode: "email_otp" });
    expect(jsonBody(adminRequest.mock.calls[2])).toMatchObject({ lock_version: 3 });
    expect(jsonBody(adminRequest.mock.calls[3])).toEqual({ lock_version: 4 });
  });

  it("uses every App user endpoint with its required password and lock body", async () => {
    await appAdminApi.members("app/a", { q: "sam", status: "active", page: 1, page_size: 25 });
    await appAdminApi.createMember("app/a", { email: "sam@example.test", display_name: "Sam", locale: "en-US", password: "correct-horse-battery" });
    await appAdminApi.updateMember("app/a", "user/a", { display_name: "Sam C", lock_version: 2 });
    await appAdminApi.memberAction("app/a", "user/a", "enable", 3);
    await appAdminApi.memberAction("app/a", "user/a", "disable", 4);
    await appAdminApi.memberAction("app/a", "user/a", "unlock", 5);
    await appAdminApi.memberAction("app/a", "user/a", "revoke-sessions", 6);
    await appAdminApi.resetMemberPassword("app/a", "user/a", "correct-horse-battery", 7);
    expect(requestCalls()).toEqual([
      ["/apps/app%2Fa/users?q=sam&status=active&page=1&page_size=25", undefined], ["/apps/app%2Fa/users", "POST"], ["/apps/app%2Fa/users/user%2Fa", "PATCH"], ["/apps/app%2Fa/users/user%2Fa/enable", "POST"], ["/apps/app%2Fa/users/user%2Fa/disable", "POST"], ["/apps/app%2Fa/users/user%2Fa/unlock", "POST"], ["/apps/app%2Fa/users/user%2Fa/sessions/revoke", "POST"], ["/apps/app%2Fa/users/user%2Fa/reset-password", "POST"],
    ]);
    expect(jsonBody(adminRequest.mock.calls[1])).toMatchObject({ email: "sam@example.test", password: "correct-horse-battery" });
    expect(jsonBody(adminRequest.mock.calls[2])).toEqual({ display_name: "Sam C", lock_version: 2 });
    expect(jsonBody(adminRequest.mock.calls[6])).toEqual({ lock_version: 6 });
    expect(jsonBody(adminRequest.mock.calls[7])).toEqual({ new_password: "correct-horse-battery", lock_version: 7 });
  });

  it("keeps markdown and blocks bodies while using page slugs for page actions", async () => {
    const input = { slug: "privacy-policy", page_type: "privacy-policy" as const, publish: false, lock_version: 5, translations: {
      "zh-CN": { title: "隐私", body_format: "blocks" as const, body: [{ type: "paragraph" as const, text: "内容" }] },
      "en-US": { title: "Privacy", body_format: "markdown" as const, body: "# Privacy" },
    } } satisfies AppPageInput;
    await appAdminApi.pages("app/a", { q: "privacy", status: "draft", page: 1, page_size: 10 });
    await appAdminApi.createPage("app/a", input);
    await appAdminApi.updatePage("app/a", "privacy-policy", input);
    await appAdminApi.publishPage("app/a", "privacy-policy", 6);
    await appAdminApi.deletePage("app/a", "privacy-policy", 7);
    expect(requestCalls()).toEqual([
      ["/apps/app%2Fa/content/pages?q=privacy&status=draft&page=1&page_size=10", undefined], ["/apps/app%2Fa/content/pages", "POST"], ["/apps/app%2Fa/content/pages/privacy-policy", "PATCH"], ["/apps/app%2Fa/content/pages/privacy-policy/publish", "POST"], ["/apps/app%2Fa/content/pages/privacy-policy?lock_version=7", "DELETE"],
    ]);
    expect((jsonBody(adminRequest.mock.calls[1])["translations"] as Record<string, unknown>)["zh-CN"]).toEqual({ title: "隐私", body_format: "blocks", body: [{ type: "paragraph", text: "内容" }] });
    expect(jsonBody(adminRequest.mock.calls[2])["lock_version"]).toBe(5);
    expect(jsonBody(adminRequest.mock.calls[3])).toEqual({ lock_version: 6 });
  });
});
