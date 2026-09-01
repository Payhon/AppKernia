import { describe, expect, it } from "vitest";
import { publicPageURL, previewChannel, previewMessage } from "./public-page";

const app = "00000000-0000-4000-8000-000000000001";
const base = `https://public.example/h5/apps/${app}`;
describe("public document URLs", () => {
  it("preserves document links and replaces the selected language", () => {
    expect(publicPageURL(`${base}/download?lang=zh-CN`, "en-US")).toBe(`${base}/download?lang=en-US`);
    expect(publicPageURL(`http://localhost:8080/s/article?app_id=${app}`, "unknown")).toContain("lang=zh-CN");
  });
  it.each([undefined, "javascript:alert(1)", "https://public.example/admin", `${base}/apk`, `${base}/download?format=qr`, "http://public.example/h5/apps/" + app + "/download", `https://name:password@public.example/h5/apps/${app}/download`, "https://public.example/s/article"])("rejects unsafe or non-document input %s", raw => {
    expect(publicPageURL(raw, "zh-CN")).toBeNull();
  });
});
describe("preview message boundary", () => {
  const frame = {} as Window;
  const event = { origin: "https://public.example", source: frame, data: { channel: previewChannel, type: "ready", loadId: "current" } } as MessageEvent<unknown>;
  it("accepts only matching source, origin, protocol and current load", () => {
    expect(previewMessage(event, frame, event.origin, "current")).toBe("ready");
    expect(previewMessage(event, {} as Window, event.origin, "current")).toBeNull();
    expect(previewMessage(event, frame, "https://evil.example", "current")).toBeNull();
    expect(previewMessage(event, frame, event.origin, "previous")).toBeNull();
    expect(previewMessage({ origin: event.origin, source: event.source, data: { channel: previewChannel, type: "navigate", loadId: "current", url: "https://evil.example" } } as MessageEvent<unknown>, frame, event.origin, "current")).toBeNull();
  });
});
