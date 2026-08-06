export const DICTIONARY_COLOR_PRESETS = [
  { key: "primary", value: "#1677ff" },
  { key: "success", value: "#087a68" },
  { key: "warning", value: "#a65a00" },
  { key: "danger", value: "#d40000" },
  { key: "purple", value: "#722ed1" },
  { key: "cyan", value: "#08979c" },
  { key: "neutral", value: "#666666" },
] as const;

export const DICTIONARY_STYLE_PRESETS = [
  { key: "primary", value: "ak-dictionary-style-primary" },
  { key: "success", value: "ak-dictionary-style-success" },
  { key: "warning", value: "ak-dictionary-style-warning" },
  { key: "danger", value: "ak-dictionary-style-danger" },
  { key: "info", value: "ak-dictionary-style-info" },
  { key: "neutral", value: "ak-dictionary-style-neutral" },
] as const;

const previewableStyleClasses = new Set<string>(
  DICTIONARY_STYLE_PRESETS.map((preset) => preset.value),
);

export function previewableDictionaryStyleClass(value: string) {
  return previewableStyleClasses.has(value) ? value : undefined;
}

export function isPreviewableDictionaryColor(value: string) {
  return /^#[\da-f]{3,4}([\da-f]{3,4})?$/i.test(value.trim());
}
