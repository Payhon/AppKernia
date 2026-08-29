// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { useState } from "react";

import { AkModal, type AkModalSize } from "./AkModal";

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false, media: query, onchange: null,
      addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
    })),
  });
  Object.defineProperty(HTMLElement.prototype, "setPointerCapture", { configurable: true, value: vi.fn() });
  Object.defineProperty(HTMLElement.prototype, "hasPointerCapture", { configurable: true, value: vi.fn(() => false) });
  Object.defineProperty(HTMLElement.prototype, "releasePointerCapture", { configurable: true, value: vi.fn() });
});

afterEach(cleanup);

function ResizableModal({ disabled = false, onResize }: { disabled?: boolean; onResize?: (size: AkModalSize) => void }) {
  const [size, setSize] = useState<AkModalSize>({ width: 640, height: 480 });
  return <AkModal
    open
    title="Resizable dialog"
    footer={null}
    resizable={{
      ...size,
      minWidth: 480,
      minHeight: 360,
      disabled,
      onResize: (next) => { setSize(next); onResize?.(next); },
    }}
    resizeHandleLabel="Resize dialog"
  >Content</AkModal>;
}

describe("AkModal", () => {
  it("exposes reusable pointer and keyboard resizing without an icon", () => {
    const onResize = vi.fn();
    render(<ResizableModal onResize={onResize} />);

    const dialog = screen.getByRole("dialog", { name: "Resizable dialog" });
    const handle = screen.getByRole<HTMLButtonElement>("button", { name: "Resize dialog" });
    expect(handle.childElementCount).toBe(0);
    expect(handle.dataset["resizing"]).toBe("false");

    fireEvent.pointerDown(handle, { button: 0, pointerId: 7, clientX: 640, clientY: 480 });
    expect(handle.dataset["resizing"]).toBe("true");
    fireEvent.pointerMove(handle, { pointerId: 7, clientX: 672, clientY: 496 });
    fireEvent.pointerUp(handle, { pointerId: 7, clientX: 672, clientY: 496 });
    expect(handle.dataset["resizing"]).toBe("false");
    expect(onResize).toHaveBeenCalledWith({ width: 672, height: 496 });
    expect(dialog.style.width).toBe("672px");

    fireEvent.keyDown(handle, { key: "ArrowLeft" });
    expect(onResize).toHaveBeenLastCalledWith({ width: 656, height: 496 });
  });

  it("does not render the resize affordance while disabled", () => {
    render(<ResizableModal disabled />);
    expect(screen.queryByRole("button", { name: "Resize dialog" })).toBeNull();
  });
});
