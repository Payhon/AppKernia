import { afterEach, describe, expect, it, vi } from "vitest";
import { authSession } from "../auth/store";
import { mobileReleasesApi } from "./api";
import { compareSemver, defaultMobileReleaseInput, mobileReleaseInputSchema, mobileReleaseInputSchemaFor, releaseCapabilityError } from "./model";

const response = (lockVersion: number) => new Response(JSON.stringify({
  code: "OK",
  message: "OK",
  data: {
    id: "123e4567-e89b-12d3-a456-426614174000",
    platform: "android",
    current_version: "2.0.0",
    minimum_version: "1.5.0",
    upgrade_url: "https://example.test/app",
    release_notes: { "zh-CN": "更新说明", "en-US": "Release notes" },
    active: true,
    lock_version: lockVersion,
    updated_at: "2026-08-05T00:00:00Z",
  },
  request_id: "release-test",
}), { status: 200, headers: { "Content-Type": "application/json" } });

describe("mobile release model", () => {
  afterEach(() => { vi.restoreAllMocks(); });

  it("validates core SemVer and rejects a minimum newer than current", () => {
    expect(compareSemver("2.0.0", "1.99.99")).toBeGreaterThan(0);
    const input = { ...defaultMobileReleaseInput("wgt"), version: "1.4.0", minimum_native_version: "1.5.0", titles: { "zh-CN": "更新", "en-US": "Update" }, contents: { "zh-CN": "说明", "en-US": "Notes" } };
    expect(mobileReleaseInputSchema.safeParse(input).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, version: "v1.5.0" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, version: "1.5.0-rc.1" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, version: "1.5.0+build.1" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, version: "1.5.0", external_url: "http://example.test/app" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, version: "1.5.0", publish_now: true, external_url: "" }).success).toBe(false);
  });

  it("allows an incomplete draft but requires bilingual content for immediate publishing", () => {
    const draft = defaultMobileReleaseInput();
    expect(mobileReleaseInputSchema.safeParse(draft).success).toBe(true);
    expect(mobileReleaseInputSchema.safeParse({ ...draft, publish_now: true, external_url: "https://example.test/app" }).success).toBe(false);
  });

  it("enforces uni-app x and native delivery capabilities", () => {
    expect(mobileReleaseInputSchemaFor("uni_app_x").safeParse(defaultMobileReleaseInput("wgt")).success).toBe(false);
    expect(mobileReleaseInputSchemaFor("uni_app").safeParse(defaultMobileReleaseInput("wgt")).success).toBe(true);
    const iosInternal = { ...defaultMobileReleaseInput(), platforms: ["ios"] as ["ios"], source_type: "internal" as const };
    expect(mobileReleaseInputSchemaFor("uni_app_x").safeParse(iosInternal).success).toBe(false);
    expect(releaseCapabilityError("uni_app_x", { package_type: "wgt", platforms: ["android"], package_file_id: null })).toBe("unsupported_package_type");
    expect(releaseCapabilityError("uni_app_x", { package_type: "native_app", platforms: ["ios"], package_file_id: "123e4567-e89b-12d3-a456-426614174000" })).toBe("unsupported_delivery_mode");
  });

  it("sends lock_version when updating and maps the incremented response", async () => {
    const request = vi.spyOn(authSession, "adminRequest").mockResolvedValueOnce(response(5));
    const input = { ...defaultMobileReleaseInput(), version: "2.0.0", external_url: "https://example.test/app", publish_now: true, lock_version: 4, titles: { "zh-CN": "更新", "en-US": "Update" }, contents: { "zh-CN": "更新说明", "en-US": "Release notes" } };
    const updated = await mobileReleasesApi.update("123e4567-e89b-12d3-a456-426614174001", "123e4567-e89b-12d3-a456-426614174000", input);
    expect(updated.lock_version).toBe(5);
    const body = request.mock.calls[0]?.[1]?.body;
    expect(JSON.parse(body as string)).toMatchObject({ lock_version: 4, package_type: "native_app", platforms: ["android"] });
  });
});
