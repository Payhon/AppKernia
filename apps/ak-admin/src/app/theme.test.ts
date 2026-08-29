import { describe, expect, it } from "vitest";

import { adminTheme } from "./theme";

describe("adminTheme popup selection contrast", () => {
  it("uses pale selected surfaces with dark text for Select and light Menu popups", () => {
    expect(adminTheme.components?.Select).toMatchObject({
      optionActiveBg: "#F5F7FA",
      optionSelectedBg: "#EAF2FF",
      optionSelectedColor: "#173E7A",
    });
    expect(adminTheme.components?.Menu).toMatchObject({
      itemHoverBg: "#F5F7FA",
      itemSelectedBg: "#EAF2FF",
      itemSelectedColor: "#173E7A",
    });
  });
});
