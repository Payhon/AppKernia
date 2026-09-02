import { describe, expect, it } from "vitest";

import { loginProviderCodes } from "./model";
import { loginProviderGuides } from "./provider-guides";

const officialHosts = new Set([
  "open.weixin.qq.com",
  "doc.dcloud.net.cn",
  "github.com",
  "docs.github.com",
  "developer.apple.com",
  "console.cloud.google.com",
  "developer.android.com",
]);

describe("login provider application guides", () => {
  it("covers every compiled provider exactly once in stable order", () => {
    expect(loginProviderGuides.map((guide) => guide.provider)).toEqual(loginProviderCodes);
    expect(new Set(loginProviderGuides.map((guide) => guide.provider)).size).toBe(loginProviderCodes.length);
  });

  it("uses only bounded official HTTPS resources", () => {
    for (const guide of loginProviderGuides) {
      expect(guide.stepCount).toBe(4);
      expect(guide.fieldKeys.length).toBeGreaterThan(0);
      expect(guide.verifiedAt).toMatch(/^\d{4}-\d{2}-\d{2}$/);
      for (const link of guide.links) {
        const url = new URL(link.url);
        expect(url.protocol).toBe("https:");
        expect(officialHosts.has(url.hostname)).toBe(true);
      }
    }
  });
});
