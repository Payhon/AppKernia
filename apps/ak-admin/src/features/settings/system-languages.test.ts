import { describe, expect, it } from "vitest";

import type { ResolvedDictionary } from "../../generated/api/types.gen";
import {
  findFirstInvalidLanguage,
  parseSystemLanguageDictionary,
  preferredSystemLanguage,
} from "./system-languages";

function dictionary(): ResolvedDictionary {
  return {
    code: "system.language",
    extension_policy: "fixed",
    items: [
      {
        extra: {},
        is_default: true,
        label: "简体中文",
        value: "zh-CN",
      },
      {
        extra: {},
        is_default: false,
        label: "English",
        value: "en-US",
      },
    ],
    locale: "zh-CN",
  };
}

describe("system languages", () => {
  it("preserves dictionary order and resolves the preferred language", () => {
    const languages = parseSystemLanguageDictionary(dictionary());

    expect(languages.map((language) => language.value)).toEqual([
      "zh-CN",
      "en-US",
    ]);
    expect(preferredSystemLanguage(languages, "en-US")).toBe("en-US");
    expect(
      findFirstInvalidLanguage(languages, [
        { path: ["translations", "en-US", "title"] },
      ]),
    ).toBe("en-US");
  });

  it("rejects missing, duplicated, or unsupported protocol languages", () => {
    const missing = dictionary();
    missing.items = missing.items.slice(0, 1);
    expect(() => parseSystemLanguageDictionary(missing)).toThrow(
      "INCOMPLETE_SYSTEM_LANGUAGE_DICTIONARY",
    );

    const unsupported = dictionary();
    unsupported.items[1] = {
      extra: {},
      is_default: false,
      label: "日本語",
      value: "ja-JP",
    };
    expect(() => parseSystemLanguageDictionary(unsupported)).toThrow(
      "UNSUPPORTED_SYSTEM_LANGUAGE",
    );
  });
});
