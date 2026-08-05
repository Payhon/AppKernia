import { afterEach, describe, expect, it, vi } from "vitest";
import { authSession } from "../auth/store";
import { mobileReleasesApi } from "./api";
import { compareSemver, defaultMobileReleaseInput, mobileReleaseInputSchema } from "./model";

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
    const input = { ...defaultMobileReleaseInput(), current_version: "1.4.0", minimum_version: "1.5.0", release_notes: { "zh-CN": "说明", "en-US": "Notes" } };
    expect(mobileReleaseInputSchema.safeParse(input).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, current_version: "v1.5.0" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, current_version: "1.5.0-rc.1" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, current_version: "1.5.0+build.1" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, current_version: "1.5.0", upgrade_url: "http://example.test/app" }).success).toBe(false);
    expect(mobileReleaseInputSchema.safeParse({ ...input, current_version: "1.5.0", active: true, upgrade_url: "" }).success).toBe(false);
  });

  it("sends lock_version when updating and maps the incremented response", async () => {
    const request = vi.spyOn(authSession, "adminRequest").mockResolvedValueOnce(response(5));
    const input = { ...defaultMobileReleaseInput(), current_version: "2.0.0", minimum_version: "1.5.0", upgrade_url: "https://example.test/app", active: true, lock_version: 4, release_notes: { "zh-CN": "更新说明", "en-US": "Release notes" } };
    const updated = await mobileReleasesApi.update("123e4567-e89b-12d3-a456-426614174000", input);
    expect(updated.lock_version).toBe(5);
    const body = request.mock.calls[0]?.[1]?.body;
    expect(JSON.parse(body as string)).toMatchObject({ lock_version: 4, platform: "android" });
  });
});
