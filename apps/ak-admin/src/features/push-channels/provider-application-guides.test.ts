import { describe, expect, it } from "vitest";
import { pushProviders } from "./model";
import { pushProviderApplicationGuides } from "./provider-application-guides";

const officialHosts = new Set([
  "developer.apple.com",
  "console.firebase.google.com",
  "firebase.google.com",
  "developer.huawei.com",
  "developer.honor.com",
  "dev.mi.com",
  "open.oppomobile.com",
  "developers.vivo.com",
  "open.flyme.cn",
  "open-res.flyme.cn",
]);

describe("push provider application guides", () => {
  it("covers every writable provider exactly once in stable order", () => {
    expect(pushProviderApplicationGuides.map((guide) => guide.provider)).toEqual(pushProviders);
    expect(new Set(pushProviderApplicationGuides.map((guide) => guide.provider)).size).toBe(pushProviders.length);
  });

  it("uses HTTPS resources hosted only by the providers", () => {
    for (const guide of pushProviderApplicationGuides) {
      expect(guide.fieldKeys.length).toBeGreaterThan(0);
      expect(guide.links.length).toBeGreaterThanOrEqual(2);
      for (const link of guide.links) {
        const url = new URL(link.url);
        expect(url.protocol).toBe("https:");
        expect(officialHosts.has(url.hostname)).toBe(true);
      }
    }
  });
});
