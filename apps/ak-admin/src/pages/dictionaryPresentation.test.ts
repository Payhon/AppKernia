import { describe, expect, it } from "vitest";
import type {
  AdminDictionaryItem,
  AdminDictionaryType,
} from "../generated/api/types.gen";
import {
  dictionaryCategoryCode,
  findTenantOverride,
  groupDictionaryTypes,
} from "./dictionaryPresentation";

const type = (id: string, code: string): AdminDictionaryType =>
  ({ id, code }) as AdminDictionaryType;
const item = (
  id: string,
  itemValue: string,
  locale: "zh-CN" | "en-US" | null,
  tenantId: string | null,
): AdminDictionaryItem =>
  ({
    id,
    item_value: itemValue,
    locale,
    tenant_id: tenantId,
    is_locked: tenantId === null,
  }) as AdminDictionaryItem;

describe("dictionary presentation", () => {
  it("groups dictionary types by the stable code namespace", () => {
    expect(dictionaryCategoryCode("storage.driver")).toBe("storage");
    expect(dictionaryCategoryCode("legacy")).toBe("custom");
    expect(
      groupDictionaryTypes(
        [
          type("1", "notification.sms.template_event"),
          type("2", "storage.driver"),
          type("3", "notification.email.template_event"),
        ],
        (code) => code,
      ).map((category) => [category.code, category.types.length]),
    ).toEqual([
      ["notification", 2],
      ["storage", 1],
    ]);
  });

  it("routes built-in editing to an existing same-locale tenant override", () => {
    const builtin = item("builtin", "cos", "zh-CN", null);
    const wrongLocale = item("english", "cos", "en-US", "tenant-1");
    const override = item("override", "cos", "zh-CN", "tenant-1");
    expect(findTenantOverride(builtin, [builtin, wrongLocale, override])).toBe(
      override,
    );
    expect(findTenantOverride(builtin, [builtin, wrongLocale])).toBeUndefined();
    expect(findTenantOverride(override, [builtin, override])).toBe(override);
  });
});
