import { describe, expect, it } from "vitest";

import {
  DICTIONARY_COLOR_PRESETS,
  DICTIONARY_STYLE_PRESETS,
  isPreviewableDictionaryColor,
  previewableDictionaryStyleClass,
} from "./dictionaryAppearance";

describe("dictionary appearance presets", () => {
  it("provides unique color and style values", () => {
    expect(new Set(DICTIONARY_COLOR_PRESETS.map((item) => item.value)).size).toBe(
      DICTIONARY_COLOR_PRESETS.length,
    );
    expect(new Set(DICTIONARY_STYLE_PRESETS.map((item) => item.value)).size).toBe(
      DICTIONARY_STYLE_PRESETS.length,
    );
  });

  it("only applies controlled style presets to the management UI", () => {
    expect(previewableDictionaryStyleClass("ak-dictionary-style-success")).toBe(
      "ak-dictionary-style-success",
    );
    expect(previewableDictionaryStyleClass("ant-row dangerous-custom-class")).toBeUndefined();
  });

  it("previews safe hexadecimal colors while retaining arbitrary values as text", () => {
    expect(isPreviewableDictionaryColor("#123456")).toBe(true);
    expect(isPreviewableDictionaryColor("#fff8")).toBe(true);
    expect(isPreviewableDictionaryColor("red; position: fixed")).toBe(false);
  });
});
