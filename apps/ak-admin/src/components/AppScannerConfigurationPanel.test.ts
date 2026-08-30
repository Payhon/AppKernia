import { describe, expect, it } from "vitest";

import { canonicalScannerHostPatterns, normalizeScannerHostPattern, scannerHostInputRowForServerIndex } from "./AppScannerConfigurationPanel";

describe("scanner host pattern validation", () => {
  it("normalizes case, trailing dots, duplicates, and wildcard domains", () => {
    expect(canonicalScannerHostPatterns(["Example.COM.", "*.Sub.Example.com", "example.com"]))
      .toEqual(["*.sub.example.com", "example.com"]);
  });

  it.each([
    "https://example.com",
    "example.com/path",
    "user@example.com",
    "example.com:8443",
    "127.0.0.1",
    "localhost",
    "*.com",
    "*.localhost",
    "例子.测试",
    "-bad.example.com",
  ])("rejects unsafe pattern %s", (value) => {
    expect(normalizeScannerHostPattern(value)).toBeNull();
  });

  it("maps a server error on sorted canonical values back to the edited input row", () => {
    const rows = ["z.example.com", "*.co.uk", "a.example.com"];
    const canonical = canonicalScannerHostPatterns(rows);
    expect(canonical).toEqual(["*.co.uk", "a.example.com", "z.example.com"]);
    expect(scannerHostInputRowForServerIndex(rows, canonical, 0)).toBe(1);
  });
});

describe("client configuration tab registry", () => {
  it("keeps sharing before scanner and declares separate permissions", async () => {
    const { clientConfigTabs } = await import("./AppClientConfigurationModal");
    expect(clientConfigTabs.map((tab) => tab.id)).toEqual(["share", "scanner"]);
    expect(clientConfigTabs.map((tab) => [tab.readPermission, tab.updatePermission])).toEqual([
      ["app.share_binding.read", "app.share_binding.update"],
      ["app.scanner_config.read", "app.scanner_config.update"],
    ]);
  });
});
