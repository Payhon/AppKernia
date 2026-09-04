// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { LocaleProvider } from "../shared/i18n";
import {
  AkInteractiveCaptcha,
  captchaPointFromClient,
  type AkCaptchaImage,
  type AkInteractiveCaptchaChallenge,
} from "./AkInteractiveCaptcha";

const image: AkCaptchaImage = {
  base64: "iVBORw0KGgo=",
  height: 200,
  mime_type: "image/png",
  width: 300,
};
const tile: AkCaptchaImage = {
  base64: "iVBORw0KGgo=",
  height: 40,
  mime_type: "image/png",
  width: 40,
};

function challenge(
  type: AkInteractiveCaptchaChallenge["type"],
): AkInteractiveCaptchaChallenge {
  const base = {
    captcha_id: `captcha-${type}`,
    captcha_token: `token-${type}`,
    expires_in_seconds: 300,
    image,
  };
  switch (type) {
    case "click":
      return { ...base, prompt_image: tile, required_points: 2, type };
    case "slide":
      return { ...base, initial_point: { x: 4, y: 60 }, tile_image: tile, type };
    case "drag":
      return { ...base, initial_point: { x: 4, y: 8 }, tile_image: tile, type };
    case "rotate":
      return { ...base, thumb_image: tile, type };
  }
}

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      addEventListener: vi.fn(),
      addListener: vi.fn(),
      dispatchEvent: vi.fn(),
      matches: false,
      media: query,
      onchange: null,
      removeEventListener: vi.fn(),
      removeListener: vi.fn(),
    })),
  });
  Object.defineProperties(HTMLElement.prototype, {
    releasePointerCapture: { configurable: true, value: vi.fn() },
    setPointerCapture: { configurable: true, value: vi.fn() },
  });
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("AkInteractiveCaptcha", () => {
  it("normalizes rendered coordinates to intrinsic image coordinates", () => {
    expect(
      captchaPointFromClient(
        110,
        70,
        { height: 100, left: 10, top: 20, width: 200 },
        { height: 200, width: 400 },
      ),
    ).toEqual({ x: 200, y: 100 });
    expect(
      captchaPointFromClient(
        110,
        70,
        { height: 100, left: 10, top: 20, width: 200 },
        { height: 200, width: 400 },
        { height: 40, width: 60 },
      ),
    ).toEqual({ x: 170, y: 80 });
  });

  it("supports pointer and keyboard point selection in the configured order", () => {
    const onChange = vi.fn();
    render(
      <LocaleProvider>
        <AkInteractiveCaptcha
          challenge={challenge("click")}
          value={null}
          onChange={onChange}
          onRefresh={vi.fn()}
        />
      </LocaleProvider>,
    );
    const board = screen.getByRole("group", {
      name: /点选验证图片|Point-selection verification image/,
    });
    vi.spyOn(board, "getBoundingClientRect").mockReturnValue({
      bottom: 100,
      height: 100,
      left: 0,
      right: 150,
      toJSON: () => ({}),
      top: 0,
      width: 150,
      x: 0,
      y: 0,
    });

    fireEvent.pointerDown(board, { clientX: 75, clientY: 50, pointerId: 1 });
    expect(onChange).toHaveBeenLastCalledWith(null);
    fireEvent.keyDown(board, { key: "ArrowRight" });
    fireEvent.keyDown(board, { key: "Enter" });

    expect(onChange).toHaveBeenLastCalledWith({
      points: [
        { x: 150, y: 100 },
        { x: 151, y: 100 },
      ],
      type: "click",
    });
  });

  it("emits a horizontal point from the accessible slide control", () => {
    const onChange = vi.fn();
    render(
      <LocaleProvider>
        <AkInteractiveCaptcha
          challenge={challenge("slide")}
          value={null}
          onChange={onChange}
          onRefresh={vi.fn()}
        />
      </LocaleProvider>,
    );

    fireEvent.change(
      screen.getByLabelText(/滑动拼图|Slide the puzzle piece/),
      { target: { value: "80" } },
    );

    expect(onChange).toHaveBeenLastCalledWith({
      point: { x: 80, y: 60 },
      type: "slide",
    });
  });

  it("supports pointer dragging and arrow-key refinement", () => {
    const onChange = vi.fn();
    render(
      <LocaleProvider>
        <AkInteractiveCaptcha
          challenge={challenge("drag")}
          value={null}
          onChange={onChange}
          onRefresh={vi.fn()}
        />
      </LocaleProvider>,
    );
    const board = screen.getByRole("group", {
      name: /拖拽验证区域|Drag verification area/,
    });
    vi.spyOn(board, "getBoundingClientRect").mockReturnValue({
      bottom: 100,
      height: 100,
      left: 0,
      right: 150,
      toJSON: () => ({}),
      top: 0,
      width: 150,
      x: 0,
      y: 0,
    });

    fireEvent.pointerDown(board, { clientX: 75, clientY: 50, pointerId: 7 });
    expect(onChange).toHaveBeenLastCalledWith({
      point: { x: 130, y: 80 },
      type: "drag",
    });
    fireEvent.keyDown(board, { key: "ArrowRight" });
    expect(onChange).toHaveBeenLastCalledWith({
      point: { x: 131, y: 80 },
      type: "drag",
    });
  });

  it("emits a rotation angle and has no automated accessibility violations", async () => {
    const onChange = vi.fn();
    render(
      <LocaleProvider>
        <main>
          <AkInteractiveCaptcha
            challenge={challenge("rotate")}
            value={null}
            onChange={onChange}
            onRefresh={vi.fn()}
          />
        </main>
      </LocaleProvider>,
    );

    fireEvent.change(screen.getByLabelText(/旋转角度|Rotation angle/), {
      target: { value: "90" },
    });

    expect(onChange).toHaveBeenLastCalledWith({ angle: 90, type: "rotate" });
    expect((await axe.run(document.body)).violations).toEqual([]);
  });

  it("expires the current challenge and resets for a replacement", () => {
    vi.useFakeTimers();
    const onChange = vi.fn();
    const first = { ...challenge("slide"), expires_in_seconds: 1 };
    const { rerender } = render(
      <LocaleProvider>
        <AkInteractiveCaptcha
          challenge={first}
          value={{ point: { x: 80, y: 60 }, type: "slide" }}
          onChange={onChange}
          onRefresh={vi.fn()}
        />
      </LocaleProvider>,
    );

    act(() => { vi.advanceTimersByTime(1_000); });
    expect(onChange).toHaveBeenLastCalledWith(null);
    expect(screen.getByText(/互动验证已过期|interactive verification has expired/i)).toBeTruthy();
    expect(screen.getByLabelText(/滑动拼图|Slide the puzzle piece/)).toHaveProperty("disabled", true);

    rerender(
      <LocaleProvider>
        <AkInteractiveCaptcha
          challenge={{ ...challenge("slide"), captcha_id: "captcha-replacement" }}
          value={null}
          onChange={onChange}
          onRefresh={vi.fn()}
        />
      </LocaleProvider>,
    );
    expect(screen.queryByText(/互动验证已过期|interactive verification has expired/i)).toBeNull();
    expect(screen.getByLabelText(/滑动拼图|Slide the puzzle piece/)).toHaveProperty("disabled", false);
  });
});
