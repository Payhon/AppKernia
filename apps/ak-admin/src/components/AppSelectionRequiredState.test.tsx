// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";

import { LocaleProvider } from "../shared/i18n";
import { AppSelectionRequiredState } from "./AppSelectionRequiredState";

afterEach(cleanup);

describe("AppSelectionRequiredState", () => {
  it("renders a centered status prompt without a loading indicator", async () => {
    const { container } = render(
      <LocaleProvider>
        <AppSelectionRequiredState />
      </LocaleProvider>,
    );

    expect(screen.getByRole("status").textContent).toMatch(/右上角选择一个应用|page header/i);
    expect(container.querySelector(".ant-spin")).toBeNull();
    expect(container.querySelector("table")).toBeNull();
    expect((await axe.run(container)).violations).toEqual([]);
  });
});
