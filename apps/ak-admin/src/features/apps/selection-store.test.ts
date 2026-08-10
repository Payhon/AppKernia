import { describe, expect, it, vi } from "vitest";

import {
  appSelectionStorageKey,
  readAppSelections,
  updateAppSelection,
  writeAppSelections,
  type AppSelectionStorage,
} from "./selection-store";

const tenantA = "019fc551-5b27-71d5-92e6-544b0105d3d6";
const tenantB = "119fc551-5b27-71d5-92e6-544b0105d3d6";
const appA = "219fc551-5b27-71d5-92e6-544b0105d3d6";
const appB = "319fc551-5b27-71d5-92e6-544b0105d3d6";

function storage(value: string | null): AppSelectionStorage {
  return { getItem: vi.fn(() => value), removeItem: vi.fn(), setItem: vi.fn() };
}

describe("App selection persistence", () => {
  it("reads only tenant-scoped UUID pairs", () => {
    const source = storage(JSON.stringify({ [tenantA]: appA, invalid_tenant: appB, [tenantB]: "__UNI__APPKERNIA" }));
    expect(readAppSelections(source)).toEqual({ [tenantA]: appA });
  });

  it("fails closed for corrupt or unavailable storage", () => {
    expect(readAppSelections(storage("not-json"))).toEqual({});
    expect(readAppSelections({ getItem: () => { throw new Error("blocked"); }, removeItem: vi.fn(), setItem: vi.fn() })).toEqual({});
  });

  it("updates and clears one tenant without changing another tenant", () => {
    const first = updateAppSelection({}, tenantA, appA);
    const second = updateAppSelection(first, tenantB, appB);
    expect(second).toEqual({ [tenantA]: appA, [tenantB]: appB });
    expect(updateAppSelection(second, tenantA, null)).toEqual({ [tenantB]: appB });
    expect(updateAppSelection(second, "invalid", appA)).toBe(second);
  });

  it("writes the non-sensitive UUID map and removes an empty preference", () => {
    const target = storage(null);
    writeAppSelections({ [tenantA]: appA }, target);
    expect(target.setItem).toHaveBeenCalledWith(appSelectionStorageKey, JSON.stringify({ [tenantA]: appA }));
    writeAppSelections({}, target);
    expect(target.removeItem).toHaveBeenCalledWith(appSelectionStorageKey);
  });
});
