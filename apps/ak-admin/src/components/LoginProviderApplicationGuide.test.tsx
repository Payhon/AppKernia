// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "../shared/i18n";
import { LoginProviderApplicationGuide } from "./LoginProviderApplicationGuide";

beforeAll(() => {
  class TestResizeObserver {
    observe = vi.fn();
    unobserve = vi.fn();
    disconnect = vi.fn();
  }
  Object.defineProperty(globalThis, "ResizeObserver", { configurable: true, value: TestResizeObserver });
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
  Object.defineProperty(window, "getComputedStyle", { configurable: true, value: () => ({ getPropertyValue: () => "" }) });
});

afterEach(async () => {
  cleanup();
  await i18n.changeLanguage("zh-CN");
});

beforeEach(async () => { await i18n.changeLanguage("zh-CN"); });

describe("LoginProviderApplicationGuide", () => {
  it("opens from an accessible question icon and exposes safe official links", () => {
    render(<I18nextProvider i18n={i18n}><LoginProviderApplicationGuide /></I18nextProvider>);
    fireEvent.click(screen.getByRole("button", { name: "查看第三方登录申请与接入指引" }));
    expect(screen.getByRole("dialog", { name: "第三方登录平台申请指引" })).toBeTruthy();
    expect(screen.getByRole("combobox", { name: "平台" })).toBeTruthy();
    expect(screen.getAllByRole("option")).toHaveLength(4);
    for (const link of screen.getAllByRole("link")) {
      expect(link.getAttribute("target")).toBe("_blank");
      expect(link.getAttribute("rel")).toBe("noopener noreferrer");
      expect(link.getAttribute("href")?.startsWith("https://")).toBe(true);
    }
  });

  it("switches to Google and renders the long English field guidance", async () => {
    await i18n.changeLanguage("en-US");
    render(<I18nextProvider i18n={i18n}><LoginProviderApplicationGuide /></I18nextProvider>);
    fireEvent.click(screen.getByRole("button", { name: "View provider application and integration guidance" }));
    fireEvent.change(screen.getByRole("combobox", { name: "Provider" }), { target: { value: "google" } });
    expect(screen.getByText("Create Android and Web OAuth clients")).toBeTruthy();
    expect(screen.getByText("The Web OAuth Client ID used by the server to verify the ID Token aud claim.")).toBeTruthy();
  });
});
