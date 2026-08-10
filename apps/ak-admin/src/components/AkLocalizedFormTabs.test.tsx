// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { SystemLanguagesState } from "../features/settings/system-languages";
import { LocaleProvider } from "../shared/i18n";
import { AkLocalizedFormTabs } from "./AkLocalizedFormTabs";

afterEach(cleanup);

beforeAll(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      disconnect() { return undefined; }
      observe() { return undefined; }
      unobserve() { return undefined; }
    },
  );
});

function readyState(): SystemLanguagesState {
  return {
    error: null,
    isError: false,
    isPending: false,
    isReady: true,
    languages: [
      { isDefault: true, label: "简体中文", value: "zh-CN" },
      { isDefault: false, label: "English", value: "en-US" },
    ],
    preferredLocale: "zh-CN",
    refetch: vi.fn(() => Promise.resolve({})),
  };
}

describe("AkLocalizedFormTabs", () => {
  it("uses dictionary labels, exposes language errors, and keeps tab panels mounted", async () => {
    const state = readyState();
    const { container, rerender } = render(
      <LocaleProvider>
        <AkLocalizedFormTabs
          activeLocale="zh-CN"
          errorLocales={{ "en-US": true }}
          languages={state}
          onActiveLocaleChange={() => undefined}
          renderFields={(locale) => (
            <input aria-label={`${locale} title`} defaultValue={locale} />
          )}
        />
      </LocaleProvider>,
    );

    expect(screen.getByRole("tab", { name: "简体中文" })).toBeTruthy();
    expect(
      screen.getByLabelText(/This language has fields|此语言包含/),
    ).toBeTruthy();
    rerender(
      <LocaleProvider>
        <AkLocalizedFormTabs
          activeLocale="en-US"
          errorLocales={{ "en-US": true }}
          languages={state}
          onActiveLocaleChange={() => undefined}
          renderFields={(locale) => (
            <input aria-label={`${locale} title`} defaultValue={locale} />
          )}
        />
      </LocaleProvider>,
    );
    const englishInput = screen.getByLabelText<HTMLInputElement>("en-US title");
    fireEvent.change(englishInput, { target: { value: "Preserved" } });
    rerender(
      <LocaleProvider>
        <AkLocalizedFormTabs
          activeLocale="zh-CN"
          errorLocales={{ "en-US": true }}
          languages={state}
          onActiveLocaleChange={() => undefined}
          renderFields={(locale) => (
            <input aria-label={`${locale} title`} defaultValue={locale} />
          )}
        />
      </LocaleProvider>,
    );
    rerender(
      <LocaleProvider>
        <AkLocalizedFormTabs
          activeLocale="en-US"
          errorLocales={{ "en-US": true }}
          languages={state}
          onActiveLocaleChange={() => undefined}
          renderFields={(locale) => (
            <input aria-label={`${locale} title`} defaultValue={locale} />
          )}
        />
      </LocaleProvider>,
    );
    expect(screen.getByLabelText<HTMLInputElement>("en-US title").value).toBe(
      "Preserved",
    );
    expect((await axe.run(container)).violations).toEqual([]);
  });

  it("renders a recoverable error when the dictionary is unavailable", () => {
    const refetch = vi.fn(() => Promise.resolve({}));
    render(
      <LocaleProvider>
        <AkLocalizedFormTabs
          activeLocale="zh-CN"
          languages={{
            ...readyState(),
            error: new Error("missing"),
            isError: true,
            isReady: false,
            languages: [],
            refetch,
          }}
          onActiveLocaleChange={() => undefined}
          renderFields={() => null}
        />
      </LocaleProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: /Retry|重试/ }));
    expect(refetch).toHaveBeenCalledOnce();
    expect(screen.getByRole("alert").textContent).toMatch(
      /system language list|系统语言列表/,
    );
  });
});
