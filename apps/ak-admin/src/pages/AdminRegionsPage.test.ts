import { describe, expect, it } from "vitest";

import { regionFormSchema } from "./AdminRegionsPage";

const valid = {
  code: "990100",
  parent_code: "990000",
  level: 1,
  name: "测试市",
  full_name: "测试省 / 测试市",
  postal_code: "518000",
  longitude: 114.0579,
  latitude: 22.5431,
  status: "active" as const,
};

describe("regionFormSchema", () => {
  it("accepts a direct level 1 or level 2 child with optional coordinates", () => {
    expect(regionFormSchema.safeParse(valid).success).toBe(true);
    expect(
      regionFormSchema.safeParse({
        ...valid,
        level: 2,
        longitude: null,
        latitude: null,
      }).success,
    ).toBe(true);
  });

  it("rejects root or deeper nodes, unsafe codes, and invalid coordinates", () => {
    expect(regionFormSchema.safeParse({ ...valid, level: 0 }).success).toBe(false);
    expect(regionFormSchema.safeParse({ ...valid, level: 3 }).success).toBe(false);
    expect(regionFormSchema.safeParse({ ...valid, code: "bad/code" }).success).toBe(false);
    expect(regionFormSchema.safeParse({ ...valid, longitude: 181 }).success).toBe(false);
  });
});
