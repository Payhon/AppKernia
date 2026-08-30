import { beforeEach, describe, expect, it, vi } from "vitest";
import { authSession } from "../auth/store";
import { listPushProviderConfigs, rotatePushProviderSecret, sendPushTest } from "./api";

describe("push channel API", () => {
  const request = vi.spyOn(authSession, "adminRequest");
  beforeEach(() => { request.mockReset(); request.mockResolvedValue(new Response(JSON.stringify({ data: { items: [] } }), { status: 200, headers: { "Content-Type": "application/json" } })); });

  it("scopes configuration reads by App and environment", async () => {
    await listPushProviderConfigs("018f08d0-3b00-7000-8000-000000000001", "production");
    expect(request).toHaveBeenCalledWith("/apps/018f08d0-3b00-7000-8000-000000000001/push-provider-configs?environment=production", undefined);
  });

  it("sends credential values only to the write-only rotation endpoint", async () => {
    request.mockResolvedValueOnce(new Response(JSON.stringify({ data: {} }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await rotatePushProviderSecret("018f08d0-3b00-7000-8000-000000000001", "018f08d0-3b00-7000-8000-000000000002", { lock_version: 2, values: { private_key_p8: "secret" } });
    const init = request.mock.calls[0]?.[1];
    expect(init?.method).toBe("POST");
    const body = init?.body;
    if (typeof body !== "string") throw new Error("expected a serialized JSON request body");
    expect(body).toContain("private_key_p8");
  });

  it("tests only a registered device id and never accepts a raw token", async () => {
    request.mockResolvedValueOnce(new Response(JSON.stringify({ data: { id: "018f08d0-3b00-7000-8000-000000000003", status: "pending" } }), { status: 202, headers: { "Content-Type": "application/json" } }));
    await sendPushTest("018f08d0-3b00-7000-8000-000000000001", "018f08d0-3b00-7000-8000-000000000002", { push_device_id: "018f08d0-3b00-7000-8000-000000000004", title: "Test", body: "Body" });
    const body = request.mock.calls[0]?.[1]?.body;
    if (typeof body !== "string") throw new Error("expected a serialized JSON request body");
    expect(body).not.toContain("token");
  });
});
