// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../features/login-providers/hooks", () => ({
  useAppLoginSettings: vi.fn(),
  useAppLoginSettingsMutation: vi.fn(),
  useAppLoginProviderBindingMutation: vi.fn(),
  useAppLoginProviderBindings: vi.fn(),
  useLoginProviderCatalog: vi.fn(),
  useLoginProviderConfigs: vi.fn(),
}));

import {
  useAppLoginSettings,
  useAppLoginSettingsMutation,
  useAppLoginProviderBindingMutation,
  useAppLoginProviderBindings,
  useLoginProviderCatalog,
  useLoginProviderConfigs,
} from "../features/login-providers/hooks";
import { appLoginProviderBindingsWriteSchema, loginProviderCodes, loginProviderDefinitions, type AppLoginProviderBinding, type LoginProviderCatalogItem } from "../features/login-providers/model";
import { ApiError } from "../shared/api/error";
import { i18n } from "../shared/i18n";
import {
  AppLoginProviderConfigurationPanel,
  buildLoginProviderBindingInput,
  loginProviderBindingFormValues,
  mergeLoginProviderBindingConflictValues,
} from "./AppLoginProviderConfigurationPanel";

const appId = "018f08d0-3b00-7000-8000-000000000010";
const configId = "018f08d0-3b00-7000-8000-000000000011";

const catalog = loginProviderCodes.map((providerCode): LoginProviderCatalogItem => {
  const definition = loginProviderDefinitions[providerCode];
  return {
    provider_code: providerCode,
    display_name_key: `login_providers.provider.${providerCode}`,
    icon_key: providerCode,
    authorization_kind: definition.authorizationKind,
    supported_platforms: [...definition.platforms],
    build_variants: [...definition.buildVariants],
    config_schema_version: 1,
    requires_secret: definition.secretFields.length > 0,
    fields: [
      { name: "external_client_id", location: "external_client_id", value_type: "string", required: true, secret: false, max_length: 255, help_key: definition.externalClientHelpKey },
      ...definition.publicFields.map((item) => ({ name: item.name, location: "public_config" as const, value_type: item.valueType, required: item.required, secret: false, max_length: item.maxLength, help_key: item.helpKey })),
      ...definition.secretFields.map((item) => ({ name: item.name, location: "secret" as const, value_type: item.valueType, required: true, secret: true, max_length: item.maxLength, help_key: item.helpKey })),
    ],
    help_url: "https://example.com/provider-help",
  };
});

const bindings: AppLoginProviderBinding[] = loginProviderCodes.map((providerCode, index) => ({
  id: providerCode === "wechat" ? "018f08d0-3b00-7000-8000-000000000012" : null,
  app_id: appId,
  provider_code: providerCode,
  login_provider_config_id: providerCode === "wechat" ? configId : null,
  config_name: providerCode === "wechat" ? "Retired WeChat" : null,
  config_status: providerCode === "wechat" ? "disabled" : null,
  preflight_status: providerCode === "wechat" ? "failed" : null,
  enabled: providerCode === "wechat",
  sort_order: (index + 1) * 10,
  lock_version: providerCode === "wechat" ? 7 : 0,
  updated_at: null,
}));

const mutation = { isPending: false, mutateAsync: vi.fn() };
const settingsMutation = { isPending: false, mutateAsync: vi.fn() };
const bindingRefetch = vi.fn();

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

beforeEach(async () => {
  await i18n.changeLanguage("zh-CN");
  mutation.mutateAsync.mockReset();
  settingsMutation.mutateAsync.mockReset();
  bindingRefetch.mockReset();
  vi.mocked(useAppLoginSettings).mockReturnValue({ data: { password_enabled: true, otp_enabled: false, email_otp_enabled: true, sms_otp_enabled: false, lock_version: 1, updated_at: null }, isPending: false, isError: false, refetch: vi.fn() } as never);
  vi.mocked(useAppLoginSettingsMutation).mockReturnValue(settingsMutation as never);
  vi.mocked(useLoginProviderCatalog).mockReturnValue({ data: catalog, isPending: false, isError: false, refetch: vi.fn() } as never);
  vi.mocked(useLoginProviderConfigs).mockReturnValue({ data: { items: [], page: 1, page_size: 100, total: 0 }, isPending: false, isError: false, refetch: vi.fn() } as never);
  vi.mocked(useAppLoginProviderBindings).mockReturnValue({ data: bindings, isPending: false, isError: false, refetch: bindingRefetch } as never);
  vi.mocked(useAppLoginProviderBindingMutation).mockReturnValue(mutation as never);
});

afterEach(() => { cleanup(); vi.clearAllMocks(); });

describe("AppLoginProviderConfigurationPanel", () => {
  it("keeps password enabled and saves OTP channels independently", async () => {
    settingsMutation.mutateAsync.mockResolvedValue({ password_enabled: true, otp_enabled: true, email_otp_enabled: true, sms_otp_enabled: true, lock_version: 2, updated_at: null });
    render(<I18nextProvider i18n={i18n}><AppLoginProviderConfigurationPanel appId={appId} canOpenConfigurationPage={false} canUpdate canReadLoginSettings canReadProviders={false} onDirtyChange={vi.fn()} /></I18nextProvider>);
    expect(screen.getByRole("switch", { name: "用户密码登录" }).hasAttribute("disabled")).toBe(true);
    fireEvent.click(screen.getByRole("switch", { name: "验证码登录" }));
    fireEvent.click(await screen.findByRole("switch", { name: "短信验证码" }));
    fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));
    await waitFor(() => { expect(settingsMutation.mutateAsync).toHaveBeenCalledWith({ otp_enabled: true, email_otp_enabled: true, sms_otp_enabled: true, lock_version: 1 }); });
  });

  it("lets an unavailable current binding be disabled and then cleared", async () => {
    const onDirtyChange = vi.fn();
    const { container } = render(<I18nextProvider i18n={i18n}><AppLoginProviderConfigurationPanel appId={appId} canOpenConfigurationPage canUpdate onDirtyChange={onDirtyChange} /></I18nextProvider>);

    const toggle = screen.getByRole("switch", { name: "开启 微信 登录" });
    expect(toggle.hasAttribute("disabled")).toBe(false);
    fireEvent.click(toggle);
    await waitFor(() => { expect(toggle.getAttribute("aria-checked")).toBe("false"); });

    const select = screen.getByRole("combobox", { name: "微信 平台配置" });
    const root = select.closest(".ant-select");
    if (!root) throw new Error("expected Ant Select root");
    fireEvent.mouseEnter(root);
    await waitFor(() => { expect(container.querySelector(".ant-select-clear")).not.toBeNull(); });
  });

  it("serializes an explicit clear as null plus the previous lock version", () => {
    const values = loginProviderBindingFormValues(bindings);
    const wechat = values.bindings[0];
    if (!wechat) throw new Error("expected WeChat row");
    wechat.enabled = false;
    wechat.login_provider_config_id = "";
    expect(buildLoginProviderBindingInput(values).bindings[0]).toEqual({
      provider_code: "wechat",
      login_provider_config_id: null,
      enabled: false,
      sort_order: 10,
      lock_version: 7,
    });
  });

  it("refreshes only lock versions after a conflict and keeps all editable rows", () => {
    const editable = loginProviderBindingFormValues(bindings);
    const wechat = editable.bindings[0];
    if (!wechat) throw new Error("expected WeChat row");
    wechat.login_provider_config_id = "018f08d0-3b00-7000-8000-000000000099";
    wechat.enabled = false;
    wechat.sort_order = 88;
    const latest = bindings.map((item) => item.provider_code === "wechat" ? { ...item, lock_version: 12, enabled: true, sort_order: 10 } : item);
    expect(mergeLoginProviderBindingConflictValues(editable, latest).bindings[0]).toEqual({
      provider_code: "wechat",
      login_provider_config_id: "018f08d0-3b00-7000-8000-000000000099",
      enabled: false,
      sort_order: 88,
      lock_version: 12,
    });
  });

  it("retains the disabled selection through a 409 refresh and retries with the latest lock", async () => {
    const latest = bindings.map((item) => item.provider_code === "wechat" ? { ...item, lock_version: 12 } : item);
    mutation.mutateAsync
      .mockRejectedValueOnce(new ApiError(409))
      .mockResolvedValueOnce(latest);
    bindingRefetch.mockResolvedValue({ data: latest });

    render(<I18nextProvider i18n={i18n}><AppLoginProviderConfigurationPanel appId={appId} canOpenConfigurationPage canUpdate onDirtyChange={vi.fn()} /></I18nextProvider>);
    fireEvent.click(screen.getByRole("switch", { name: "开启 微信 登录" }));
    fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));
    await waitFor(() => { expect(screen.getByText("至少一项配置已被其他管理员修改；本次请求没有部分生效，请刷新后重新确认。")).toBeTruthy(); });

    fireEvent.click(screen.getByRole("button", { name: /刷\s*新/ }));
    await waitFor(() => { expect(bindingRefetch).toHaveBeenCalledTimes(1); });
    expect(screen.getByRole("switch", { name: "开启 微信 登录" }).getAttribute("aria-checked")).toBe("false");

    fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));
    await waitFor(() => { expect(mutation.mutateAsync).toHaveBeenCalledTimes(2); });
    const retriedInput = appLoginProviderBindingsWriteSchema.parse(mutation.mutateAsync.mock.calls[1]?.[0]);
    expect(retriedInput.bindings[0]).toMatchObject({
      provider_code: "wechat",
      login_provider_config_id: configId,
      enabled: false,
      sort_order: 10,
      lock_version: 12,
    });
  });

  it("renders all four providers read-only and hides the bulk save action", () => {
    render(<I18nextProvider i18n={i18n}><AppLoginProviderConfigurationPanel appId={appId} canOpenConfigurationPage={false} canUpdate={false} onDirtyChange={vi.fn()} /></I18nextProvider>);
    expect(screen.getAllByRole("switch")).toHaveLength(4);
    expect(screen.queryByRole("button", { name: "保存" })).toBeNull();
    for (const toggle of screen.getAllByRole("switch")) expect(toggle.hasAttribute("disabled")).toBe(true);
  });
});
