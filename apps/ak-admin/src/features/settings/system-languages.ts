import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type { ResolvedDictionary } from "../../generated/api/types.gen";
import {
  supportedLocales,
  type AdminLocale,
} from "../../shared/i18n";
import { useAdminDictionary } from "./hooks";

export const SYSTEM_LANGUAGE_DICTIONARY_CODE = "system.language";

export interface SystemLanguageOption {
  isDefault: boolean;
  label: string;
  value: AdminLocale;
}

export interface SystemLanguagesState {
  error: unknown;
  isError: boolean;
  isPending: boolean;
  isReady: boolean;
  languages: readonly SystemLanguageOption[];
  preferredLocale: AdminLocale;
  refetch: () => Promise<unknown>;
}

function isAdminLocale(value: string): value is AdminLocale {
  return supportedLocales.some((locale) => locale === value);
}

export function parseSystemLanguageDictionary(
  dictionary: ResolvedDictionary,
): SystemLanguageOption[] {
  if (
    dictionary.code !== SYSTEM_LANGUAGE_DICTIONARY_CODE ||
    dictionary.extension_policy !== "fixed"
  ) {
    throw new Error("INVALID_SYSTEM_LANGUAGE_DICTIONARY");
  }

  const languages = dictionary.items.map((item) => {
    if (!isAdminLocale(item.value)) {
      throw new Error("UNSUPPORTED_SYSTEM_LANGUAGE");
    }
    return {
      isDefault: item.is_default,
      label: item.label.trim(),
      value: item.value,
    };
  });

  const values = new Set(languages.map((language) => language.value));
  const defaults = languages.filter((language) => language.isDefault);
  if (
    languages.some((language) => !language.label) ||
    languages.length !== supportedLocales.length ||
    values.size !== supportedLocales.length ||
    supportedLocales.some((locale) => !values.has(locale)) ||
    defaults.length !== 1
  ) {
    throw new Error("INCOMPLETE_SYSTEM_LANGUAGE_DICTIONARY");
  }

  return languages;
}

export function preferredSystemLanguage(
  languages: readonly SystemLanguageOption[],
  currentLocale: AdminLocale,
): AdminLocale {
  return (
    languages.find((language) => language.value === currentLocale)?.value ??
    languages.find((language) => language.isDefault)?.value ??
    languages[0]?.value ??
    "zh-CN"
  );
}

export function findFirstInvalidLanguage(
  languages: readonly SystemLanguageOption[],
  issues: readonly { path: readonly PropertyKey[] }[],
): AdminLocale | undefined {
  return languages.find((language) =>
    issues.some((issue) => issue.path.includes(language.value)),
  )?.value;
}

export function useSystemLanguages(): SystemLanguagesState {
  const { i18n } = useTranslation();
  const query = useAdminDictionary(SYSTEM_LANGUAGE_DICTIONARY_CODE);
  const parsed = useMemo(() => {
    if (!query.data) return { error: null, languages: [] };
    try {
      return {
        error: null,
        languages: parseSystemLanguageDictionary(query.data),
      };
    } catch (error) {
      return { error, languages: [] };
    }
  }, [query.data]);
  const currentLocale: AdminLocale =
    i18n.resolvedLanguage === "en-US" ? "en-US" : "zh-CN";
  const error = query.error ?? parsed.error;

  return {
    error,
    isError: Boolean(error),
    isPending: query.isPending,
    isReady: query.isSuccess && !error && parsed.languages.length > 0,
    languages: parsed.languages,
    preferredLocale: preferredSystemLanguage(parsed.languages, currentLocale),
    refetch: async () => query.refetch(),
  };
}
