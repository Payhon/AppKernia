import { describe, expect, it } from "vitest";

import { enabledCapabilities } from "./AdminServiceHealthPage";

describe("enabledCapabilities", () => {
  it("returns only compiled capabilities in stable order", () => {
    expect(
      enabledCapabilities({ worker: true, runtime_summary: true, shell: false }),
    ).toEqual(["runtime_summary", "worker"]);
  });

  it("handles modules without enabled capabilities", () => {
    expect(enabledCapabilities({ plugin_upload: false })).toEqual([]);
  });
});
