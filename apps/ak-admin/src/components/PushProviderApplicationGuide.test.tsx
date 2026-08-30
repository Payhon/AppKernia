// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { i18n } from "../shared/i18n";
import { PushProviderApplicationGuide } from "./PushProviderApplicationGuide";

beforeAll(() => {
  class TestResizeObserver {
    observe = vi.fn();
    unobserve = vi.fn();
    disconnect = vi.fn();
  }
  Object.defineProperty(globalThis, "ResizeObserver", {
    configurable: true,
    value: TestResizeObserver,
  });
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
  Object.defineProperty(window, "getComputedStyle", {
    configurable: true,
    value: () => ({ getPropertyValue: () => "" }),
  });
});

afterEach(async () => {
  cleanup();
  await i18n.changeLanguage("zh-CN");
});

beforeEach(async () => {
  await i18n.changeLanguage("zh-CN");
});

describe("PushProviderApplicationGuide", () => {
  it("opens from an accessible icon and exposes all provider tabs and official links", () => {
    render(<I18nextProvider i18n={i18n}><PushProviderApplicationGuide /></I18nextProvider>);
    fireEvent.click(screen.getByRole("button", { name: "查看推送渠道申请与对接指引" }));

    expect(screen.getByRole("dialog", { name: "推送渠道申请与对接指引" })).toBeTruthy();
    expect(screen.getAllByRole("tab")).toHaveLength(9);
    expect(screen.getByRole("tab", { name: "Apple APNs" })).toBeTruthy();
    expect(screen.getAllByRole("link")).toHaveLength(3);
    for (const link of screen.getAllByRole("link")) {
      expect(link.getAttribute("target")).toBe("_blank");
      expect(link.getAttribute("rel")).toBe("noopener noreferrer");
    }
  });

  it("switches providers and renders the English guide", async () => {
    await i18n.changeLanguage("en-US");
    render(<I18nextProvider i18n={i18n}><PushProviderApplicationGuide /></I18nextProvider>);
    fireEvent.click(screen.getByRole("button", { name: "View push channel application and integration guides" }));
    fireEvent.click(screen.getByRole("tab", { name: "vivo" }));
    expect(screen.getByText("Enable and verify vivo Push")).toBeTruthy();
    expect(screen.getByText("Service and security category (provider-approved value)")).toBeTruthy();
  });
});
