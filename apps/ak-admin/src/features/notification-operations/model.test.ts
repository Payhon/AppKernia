import { describe, expect, it } from "vitest";
import { operationsTimeRange, readOperationsFilters } from "./model";

describe("notification operations filters", () => {
  it("restores safe URL state and rejects malformed pagination", () => {
    const value = readOperationsFilters("?tab=tasks&range=7d&page=-1&page_size=999&environment=production");
    expect(value).toMatchObject({ tab: "tasks", range: "7d", page: 1, page_size: 20, environment: "production" });
  });

  it("uses an exact UTC range", () => {
    const now = new Date("2026-08-29T00:00:00.000Z");
    expect(operationsTimeRange("7d", now)).toEqual({ from: "2026-08-22T00:00:00.000Z", to: now.toISOString() });
  });
});
