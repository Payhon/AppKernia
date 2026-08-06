// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { AkCreatableSelect } from "./AkCreatableSelect";

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
  class TestResizeObserver {
    observe() {
      return undefined;
    }
    disconnect() {
      return undefined;
    }
    unobserve() {
      return undefined;
    }
  }
  Object.defineProperty(globalThis, "ResizeObserver", {
    configurable: true,
    value: TestResizeObserver,
  });
});

afterEach(cleanup);

describe("AkCreatableSelect", () => {
  it("offers a default value and presets without accessibility violations", async () => {
    const onChange = vi.fn();
    const { container } = render(
      <main>
        <AkCreatableSelect
          aria-label="Color"
          defaultLabel="Default (not set)"
          onChange={onChange}
          options={[
            {
              label: "Success green #087a68",
              searchText: "Success green #087a68",
              selectedLabel: "Success green",
              value: "#087a68",
            },
          ]}
          value=""
        />
      </main>,
    );

    expect(screen.getByText("Default (not set)")).toBeTruthy();
    fireEvent.mouseDown(screen.getByRole("combobox"));
    fireEvent.click(await screen.findByText("Success green #087a68"));
    expect(onChange).toHaveBeenLastCalledWith("#087a68");
    expect((await axe.run(container)).violations).toEqual([]);
  });

  it("creates a custom single value from keyboard input", () => {
    const onChange = vi.fn();
    render(
      <AkCreatableSelect
        aria-label="CSS class"
        defaultLabel="Default (not set)"
        onChange={onChange}
        options={[]}
        value=""
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.mouseDown(input);
    fireEvent.change(input, { target: { value: "tenant-accent" } });
    fireEvent.keyDown(input, {
      key: "Enter",
      code: "Enter",
      keyCode: 13,
      which: 13,
    });
    expect(onChange).toHaveBeenLastCalledWith("tenant-accent");
  });
});
