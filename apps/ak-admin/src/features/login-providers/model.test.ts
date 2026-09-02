import { describe, expect, it } from "vitest";

import {
  appLoginProviderBindingsSchema,
  appLoginProviderBindingsWriteSchema,
  buildLoginProviderConfigInput,
  catalogMatchesDefinition,
  emptyLoginProviderConfigForm,
  loginProviderCodes,
  loginProviderConfigActions,
  loginProviderConfigSchema,
  mergeLoginProviderConflictValues,
  type LoginProviderCatalogItem,
} from "./model";

const field = (
  name: string,
  location: "external_client_id" | "public_config" | "secret",
  secret: boolean,
  valueType: "string" | "url" | "string_array" | "pem" = "string",
) => ({ name, location, value_type: valueType, required: true, secret, max_length: 4096, help_key: `login_providers.field_help.${name}` });

const wechatCatalog: LoginProviderCatalogItem = {
  provider_code: "wechat",
  display_name_key: "login_providers.provider.wechat",
  icon_key: "wechat",
  authorization_kind: "native_code",
  supported_platforms: ["android", "ios", "harmony"],
  build_variants: ["android_google", "android_china", "ios", "harmony"],
  config_schema_version: 1,
  requires_secret: true,
  fields: [
    field("external_client_id", "external_client_id", false),
    field("android_package_name", "public_config", false),
    field("android_app_signature", "public_config", false),
    field("ios_bundle_id", "public_config", false),
    field("ios_universal_link", "public_config", false, "url"),
    field("harmony_bundle_name", "public_config", false),
    field("app_secret", "secret", true),
  ],
  help_url: "https://open.weixin.qq.com/",
};

const config = loginProviderConfigSchema.parse({
  id: "018f08d0-3b00-7000-8000-000000000001",
  name: "WeChat production",
  description: "Production identity",
  provider_code: "wechat",
  external_client_id: "wx1234567890abcdef",
  config_schema_version: 1,
  public_config: {
    android: { enabled: false },
    ios: { enabled: false },
    harmony: { enabled: false },
  },
  secret_field_names: ["app_secret"],
  has_secret: true,
  status: "draft",
  last_preflight_at: null,
  last_preflight_status: null,
  last_preflight_issues: [],
  binding_count: 0,
  lock_version: 3,
  created_at: "2026-09-01T00:00:00Z",
  updated_at: "2026-09-01T00:00:00Z",
});

describe("login provider model", () => {
  it("accepts flattened catalog descriptors and rejects platform/schema drift", () => {
    expect(catalogMatchesDefinition(wechatCatalog)).toBe(true);
    expect(catalogMatchesDefinition({ ...wechatCatalog, fields: wechatCatalog.fields.map((item) => item.name === "android_app_signature" ? { ...item, name: "signature" } : item) })).toBe(false);
    expect(catalogMatchesDefinition({ ...wechatCatalog, supported_platforms: ["android", "ios"] })).toBe(false);
  });

  it("serializes WeChat public values to the strict nested v1 shape", () => {
    const values = emptyLoginProviderConfigForm("wechat");
    values.name = "WeChat production";
    values.external_client_id = "wx1234567890abcdef";
    values.public_values = {
      android_package_name: "com.appkernia.mobile",
      android_app_signature: "ABCDEF",
      ios_bundle_id: "com.appkernia.mobile",
      ios_universal_link: "https://app.example.com/wechat/",
      harmony_bundle_name: "com.appkernia.mobile",
    };
    expect(buildLoginProviderConfigInput(values, 1)?.public_config).toEqual({
      android: { enabled: true, package_name: "com.appkernia.mobile", app_signature: "ABCDEF" },
      ios: { enabled: true, bundle_id: "com.appkernia.mobile", universal_link: "https://app.example.com/wechat/" },
      harmony: { enabled: true, bundle_name: "com.appkernia.mobile" },
    });
  });

  it("requires exactly four unique atomic bindings and disallows enabled null configs", () => {
    const bindings = loginProviderCodes.map((providerCode, index) => ({
      provider_code: providerCode,
      login_provider_config_id: "018f08d0-3b00-7000-8000-000000000001",
      enabled: true,
      sort_order: (index + 1) * 10,
      lock_version: index,
    }));
    expect(appLoginProviderBindingsWriteSchema.safeParse({ bindings }).success).toBe(true);
    expect(appLoginProviderBindingsWriteSchema.safeParse({ bindings: bindings.map((item, index) => index === 0 ? { ...item, login_provider_config_id: null } : item) }).success).toBe(false);
    expect(appLoginProviderBindingsWriteSchema.safeParse({ bindings: bindings.map((item) => ({ ...item, provider_code: "wechat" })) }).success).toBe(false);
  });

  it("parses four unbound response rows with null ids and omitted fingerprints", () => {
    const parsed = appLoginProviderBindingsSchema.parse({
      items: loginProviderCodes.map((providerCode, index) => ({
        id: null,
        app_id: "018f08d0-3b00-7000-8000-000000000002",
        provider_code: providerCode,
        login_provider_config_id: null,
        config_name: null,
        config_status: null,
        preflight_status: null,
        enabled: false,
        sort_order: (index + 1) * 10,
        lock_version: 0,
        updated_at: null,
      })),
    });
    expect(parsed.items).toHaveLength(4);
    expect(config.credential_fingerprint).toBeUndefined();
  });

  it("keeps user-editable values while adopting the latest immutable provider and lock", () => {
    const edited = emptyLoginProviderConfigForm("wechat");
    edited.name = "Unsaved edit";
    edited.description = "Unsaved description";
    edited.external_client_id = "wxabcdef1234567890";
    edited.public_values = {
      ...edited.public_values,
      android_package_name: "com.example.unsaved",
      android_app_signature: "ABCDEF",
    };
    const latest = { ...config, lock_version: 11 };
    const merged = mergeLoginProviderConflictValues(edited, latest);
    expect(merged.name).toBe("Unsaved edit");
    expect(merged.description).toBe("Unsaved description");
    expect(merged.external_client_id).toBe("wxabcdef1234567890");
    expect(merged.provider_code).toBe("wechat");
    expect(buildLoginProviderConfigInput(merged, latest.config_schema_version, latest.lock_version)).toMatchObject({
      name: "Unsaved edit",
      description: "Unsaved description",
      external_client_id: "wxabcdef1234567890",
      lock_version: 11,
      public_config: {
        android: { enabled: true, package_name: "com.example.unsaved", app_signature: "ABCDEF" },
      },
    });
  });

  it("prunes lifecycle actions by stable permission and configuration state", () => {
    expect([...loginProviderConfigActions(config, new Set())]).toEqual([]);
    expect([...loginProviderConfigActions(config, new Set(["sys.login_provider_config.update"]))]).toEqual(["edit"]);
    expect([...loginProviderConfigActions(
      { ...config, last_preflight_status: "ready" },
      new Set(["sys.login_provider_config.update", "sys.login_provider_config.rotate_secret", "sys.login_provider_config.preflight", "sys.login_provider_config.delete"]),
    )]).toEqual(["edit", "activate", "rotate_secret", "preflight", "delete"]);
  });
});
