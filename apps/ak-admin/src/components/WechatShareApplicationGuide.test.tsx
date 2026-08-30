// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "../shared/i18n";
import { WechatShareApplicationGuide } from "./WechatShareApplicationGuide";

beforeAll(() => {
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

describe("WechatShareApplicationGuide", () => {
  it("opens a five-step guide and keeps every official resource in a new browsing context", () => {
    render(<I18nextProvider i18n={i18n}><WechatShareApplicationGuide /></I18nextProvider>);

    fireEvent.click(screen.getByRole("button", { name: "查看微信开放平台申请指引" }));

    expect(screen.getByRole("dialog", { name: "如何申请微信分享 AppID？" })).toBeTruthy();
    expect(screen.getAllByRole("listitem").length).toBeGreaterThanOrEqual(10);
    const links = screen.getAllByRole("link");
    expect(links.length).toBe(8);
    for (const link of links) {
      expect(link.getAttribute("target")).toBe("_blank");
      expect(link.getAttribute("rel")).toBe("noopener noreferrer");
      expect(link.getAttribute("href")?.startsWith("https://")).toBe(true);
    }
  });

  it("renders the same independent guidance in English", async () => {
    await i18n.changeLanguage("en-US");
    render(<I18nextProvider i18n={i18n}><WechatShareApplicationGuide /></I18nextProvider>);

    fireEvent.click(screen.getByRole("button", { name: "View the WeChat Open Platform application guide" }));
    expect(screen.getByRole("dialog", { name: "How do I obtain a WeChat sharing AppID?" })).toBeTruthy();
  });
});
