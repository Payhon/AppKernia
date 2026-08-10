import { describe, expect, it } from "vitest";

import { readAppId, requiresAppSelection, resolveAppId, withSelectedApp } from "./scope";

describe("App scope routing", () => {
  it("reads only UUID application identifiers", () => {
    expect(readAppId("?app_id=019fc551-5b27-71d5-92e6-544b0105d3d6")).toBe("019fc551-5b27-71d5-92e6-544b0105d3d6");
    expect(readAppId("?app_id=__UNI__APPKERNIA")).toBeNull();
    expect(readAppId("?q=alpha")).toBeNull();
  });

  it.each([
    "/app/users",
    "/app/users/",
    "/app/upgrade-center",
    "/app/content/articles",
    "/app/content/categories",
    "/app/content/pages",
    "/system/mobile/releases",
  ])("shows the global selector on %s", (pathname) => {
    expect(requiresAppSelection(pathname)).toBe(true);
  });

  it.each(["/app/applications", "/app", "/dashboard", "/system/content/articles", "/system/mobile/profile"])(
    "does not show the global selector on %s",
    (pathname) => {
      expect(requiresAppSelection(pathname)).toBe(false);
    },
  );

  it("preserves URL filters and resets pagination when the App changes", () => {
    expect(withSelectedApp({ q: "release", package_type: "wgt", page: 4, page_size: 20 }, "019fc551-5b27-71d5-92e6-544b0105d3d6")).toEqual({
      q: "release",
      package_type: "wgt",
      page: 1,
      page_size: 20,
      app_id: "019fc551-5b27-71d5-92e6-544b0105d3d6",
    });
    expect(withSelectedApp({ q: "release", app_id: "old" }, undefined)).toEqual({ q: "release", app_id: undefined });
  });

  it("prefers an explicit URL App and otherwise restores the remembered App", () => {
    const urlAppId = "019fc551-5b27-71d5-92e6-544b0105d3d6";
    const rememberedAppId = "119fc551-5b27-71d5-92e6-544b0105d3d6";
    expect(resolveAppId(urlAppId, rememberedAppId)).toBe(urlAppId);
    expect(resolveAppId(null, rememberedAppId)).toBe(rememberedAppId);
    expect(resolveAppId(null, null)).toBeNull();
  });
});
