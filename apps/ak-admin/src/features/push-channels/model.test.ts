import { describe, expect, it } from "vitest";
import { fieldValuesSchema, providerState, testPushSchema } from "./model";

describe("push channel models", () => {
  it("requires every advertised provider field", () => {
    const schema = fieldValuesSchema(["project_id", "package_name"]);
    expect(schema.safeParse({ project_id: "ak", package_name: "com.appkernia.mobile" }).success).toBe(true);
    expect(schema.safeParse({ project_id: "ak", package_name: "" }).success).toBe(false);
  });

  it("accepts bounded service-account and private-key credentials", () => {
    const schema = fieldValuesSchema(["service_account_json"], 65_536);
    expect(schema.safeParse({ service_account_json: "x".repeat(4096) }).success).toBe(true);
    expect(schema.safeParse({ service_account_json: "x".repeat(65_537) }).success).toBe(false);
  });

  it("accepts only bounded test messages and opaque device ids", () => {
    expect(testPushSchema.safeParse({ push_device_id: "018f08d0-3b00-7000-8000-000000000001", title: "Test", body: "Hello" }).success).toBe(true);
    expect(testPushSchema.safeParse({ push_device_id: "raw-token", title: "", body: "Hello" }).success).toBe(false);
  });

  it("shows preflight-ready drafts separately from active configs", () => {
    expect(providerState(null)).toBe("unconfigured");
    expect(providerState({ status: "draft", last_preflight_status: "ready" } as never)).toBe("ready");
    expect(providerState({ status: "active" } as never)).toBe("active");
  });
});
