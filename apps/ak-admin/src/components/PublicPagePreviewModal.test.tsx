// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { PublicPagePreviewModal } from "./PublicPagePreviewModal";
import { previewChannel } from "../shared/public-page";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en-US" } }) }));
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", { configurable: true, value: vi.fn(() => ({ matches: false, addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn() })) });
  vi.stubGlobal("ResizeObserver", class { observe() { /* jsdom has no layout */ } disconnect() { /* no observer resource */ } unobserve() { /* no observer resource */ } });
});
afterEach(() => { cleanup(); vi.useRealTimers(); vi.restoreAllMocks(); });
const url = "https://public.example/h5/apps/00000000-0000-4000-8000-000000000001/download";
describe("public page preview", () => {
  it("creates no document while closed and removes the document on close", () => {
    const view = render(<PublicPagePreviewModal open={false} url={url} title="Fixture" onClose={vi.fn()} />);
    expect(document.querySelector("iframe")).toBeNull();
    view.rerender(<PublicPagePreviewModal open url={url} title="Fixture" onClose={vi.fn()} />);
    expect(document.querySelector("iframe")?.src).toBe(`${url}?lang=en-US`);
    view.rerender(<PublicPagePreviewModal open={false} url={url} title="Fixture" onClose={vi.fn()} />);
    expect(document.querySelector("iframe")).toBeNull();
  });
  it("does not mistake load for readiness, refreshes with a new frame and validates close messages", () => {
    const close = vi.fn();
    render(<PublicPagePreviewModal open url={url} title="Fixture" onClose={close} />);
    const frame = document.querySelector("iframe");
    if (!frame?.contentWindow) throw new Error("missing preview");
    const post = vi.spyOn(frame.contentWindow, "postMessage");
    fireEvent.load(frame);
    expect(screen.queryByText("apps.public_web.preview_loading")).not.toBeNull();
    const init = post.mock.calls[0]?.[0] as { loadId: string };
    const emit = (type: string, origin = "https://public.example", loadId = init.loadId) => act(() => window.dispatchEvent(new MessageEvent("message", { source: frame.contentWindow, origin, data: { channel: previewChannel, loadId, type } })));
    void emit("ready", "https://evil.example");
    expect(screen.queryByText("apps.public_web.preview_loading")).not.toBeNull();
    void emit("ready");
    expect(screen.queryByText("apps.public_web.preview_loading")).toBeNull();
    void emit("close", "https://public.example", "old-load");
    expect(close).not.toHaveBeenCalled();
    void emit("close");
    expect(close).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "apps.public_web.preview_refresh" }));
    expect(document.querySelector("iframe")).not.toBe(frame);
    expect(document.querySelector("iframe")?.src).toBe(`${url}?lang=en-US`);
  });
  it("shows a retry and independent link after the readiness deadline", () => {
    vi.useFakeTimers();
    render(<PublicPagePreviewModal open url={url} title="Fixture" onClose={vi.fn()} />);
    act(() => { vi.advanceTimersByTime(10001); });
    expect(screen.queryByText("apps.public_web.preview_timeout")).not.toBeNull();
    expect(screen.getByRole("link", { name: "apps.public_web.view" }).getAttribute("href")).toBe(`${url}?lang=en-US`);
  });
  it("shows a manually selectable URL when clipboard permission is denied", async () => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: vi.fn().mockRejectedValue(new Error("denied")) } });
    render(<PublicPagePreviewModal open url={url} title="Fixture" onClose={vi.fn()} />);
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "apps.public_web.copy" })); await Promise.resolve(); });
    expect(screen.getByRole<HTMLInputElement>("textbox", { name: "apps.public_web.copy" }).value).toBe(`${url}?lang=en-US`);
  });
});
