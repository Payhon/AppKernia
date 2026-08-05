// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { useTranslation } from "react-i18next";

import type { AdminConfigItem } from "../generated/api/types.gen";
import { LocaleProvider } from "../shared/i18n";
import {
  AkConfigValueField,
  configDraftValue,
  parseConfigDraftValue,
  validateConfigDraftValue,
} from "./AkConfigValueField";

function item(
  value_type: AdminConfigItem["value_type"],
  value: unknown,
  validation_schema: Record<string, unknown> = {},
): AdminConfigItem {
  return {
    id: `setting-${value_type}`,
    tenant_id: "tenant-test",
    module_code: "core",
    config_group: "basic",
    config_key: `test.${value_type}`,
    display_name: value_type,
    value_type,
    value,
    default_value: null,
    is_secret: false,
    secret_configured: false,
    secret_key_version: null,
    is_public: false,
    validation_schema,
    description: "",
    sort_order: 1,
    status: "active",
    version: 1,
    is_locked: false,
    created_at: "2026-08-05T00:00:00Z",
    updated_at: "2026-08-05T00:00:00Z",
  };
}

function FieldHarness({ config }: { config: AdminConfigItem }) {
  const { t } = useTranslation();
  return (
    <AkConfigValueField
      disabled={false}
      item={config}
      label="Test setting"
      t={t}
      value={configDraftValue(config)}
      onChange={() => undefined}
    />
  );
}

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

describe("AkConfigValueField", () => {
  it("renders a labelled definition-driven control without accessibility violations", async () => {
    render(
      <LocaleProvider>
        <main>
          <FieldHarness config={item("string", "AppKernia", { maxLength: 80 })} />
        </main>
      </LocaleProvider>,
    );
    const input = screen.getByLabelText("Test setting");
    expect(input.getAttribute("maxlength")).toBe("80");
    fireEvent.change(input, { target: { value: "AK" } });
    expect((await axe.run(document.body)).violations).toEqual([]);
  });

  it("parses JSON and applies numeric definition limits", () => {
    const json = item("json", { enabled: true });
    expect(parseConfigDraftValue(json, '{"enabled":false}')).toEqual({
      enabled: false,
    });
    const number = item("integer", 5, { minimum: 2, maximum: 10 });
    const translate = ((key: string) => key) as ReturnType<
      typeof useTranslation
    >["t"];
    expect(validateConfigDraftValue(number, 1, translate)).toBe(
      "settings.configs.form.validation.minimum",
    );
    expect(validateConfigDraftValue(number, 6, translate)).toBeUndefined();
  });
});
