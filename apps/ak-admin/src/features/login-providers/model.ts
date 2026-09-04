import { z } from "zod";

import type {
  LoginProviderBinding as GeneratedLoginProviderBinding,
  LoginProviderBindingBulkRequest,
  LoginProviderBindingInput,
  LoginProviderCode as GeneratedLoginProviderCode,
  LoginProviderConfig as GeneratedLoginProviderConfig,
  LoginProviderConfigRequest,
  LoginProviderConfigUpdateRequest,
  LoginProviderPlatform as GeneratedLoginProviderPlatform,
  LoginProviderPublicConfig,
  LoginProviderSecretRequestWritable,
} from "../../generated/api/types.gen";

export const loginProviderCodes = ["wechat", "github", "apple", "google"] as const satisfies readonly GeneratedLoginProviderCode[];
export const loginProviderCodeSchema = z.enum(loginProviderCodes);
export type LoginProviderCode = GeneratedLoginProviderCode;

export const loginProviderPlatforms = ["android", "ios", "harmony"] as const satisfies readonly GeneratedLoginProviderPlatform[];
export const loginProviderPlatformSchema = z.enum(loginProviderPlatforms);
export type LoginProviderPlatform = GeneratedLoginProviderPlatform;

export const loginProviderBuildVariants = ["android_google", "android_china", "ios", "harmony"] as const;
export const loginProviderBuildVariantSchema = z.enum(loginProviderBuildVariants);

export const loginProviderAuthorizationKinds = ["native_code", "native_id_token", "browser_ticket"] as const;
export const loginProviderAuthorizationKindSchema = z.enum(loginProviderAuthorizationKinds);

export const loginProviderConfigStatusSchema = z.enum(["draft", "active", "disabled"]);
export type LoginProviderConfigStatus = z.infer<typeof loginProviderConfigStatusSchema>;
export const loginProviderPreflightStatusSchema = z.enum(["ready", "failed"]);
export type LoginProviderPreflightStatus = z.infer<typeof loginProviderPreflightStatusSchema>;

const loginProviderPlatformConfigSchema = z.object({
  enabled: z.boolean(),
  package_name: z.string().optional(),
  app_signature: z.string().optional(),
  bundle_id: z.string().optional(),
  universal_link: z.url().optional(),
  bundle_name: z.string().optional(),
});

const loginProviderPublicConfigSchema = z.union([
  z.object({
    android: loginProviderPlatformConfigSchema,
    ios: loginProviderPlatformConfigSchema,
    harmony: loginProviderPlatformConfigSchema,
  }),
  z.object({ app_return_uri: z.url() }),
  z.object({ team_id: z.string(), key_id: z.string() }),
  z.object({ android_package_name: z.string(), android_certificate_sha256: z.array(z.string()) }),
]).transform((value): LoginProviderPublicConfig => value as LoginProviderPublicConfig);

const catalogFieldSchema = z.object({
  name: z.string().min(1).max(128),
  location: z.enum(["external_client_id", "public_config", "secret"]),
  value_type: z.enum(["string", "url", "string_array", "pem"]),
  required: z.boolean(),
  secret: z.boolean(),
  max_length: z.number().int().positive().max(1_048_576),
  help_key: z.string().min(1).max(255),
});

export const loginProviderCatalogItemSchema = z.object({
  provider_code: loginProviderCodeSchema,
  display_name_key: z.string().min(1).max(255),
  icon_key: z.string().min(1).max(64),
  authorization_kind: loginProviderAuthorizationKindSchema,
  supported_platforms: z.array(loginProviderPlatformSchema),
  build_variants: z.array(loginProviderBuildVariantSchema),
  config_schema_version: z.literal(1),
  requires_secret: z.boolean(),
  fields: z.array(catalogFieldSchema),
  help_url: z.url().refine((value) => value.startsWith("https://")),
});
export type LoginProviderCatalogItem = z.infer<typeof loginProviderCatalogItemSchema>;

const rawLoginProviderConfigSchema = z.object({
  id: z.uuid(),
  name: z.string().min(1).max(160),
  description: z.string().max(2000),
  provider_code: loginProviderCodeSchema,
  external_client_id: z.string().min(1).max(255),
  config_schema_version: z.literal(1),
  public_config: loginProviderPublicConfigSchema,
  secret_field_names: z.array(z.string().min(1).max(128)),
  has_secret: z.boolean(),
  credential_fingerprint: z.string().max(255).nullable().optional(),
  status: loginProviderConfigStatusSchema,
  last_preflight_at: z.string().nullable(),
  last_preflight_status: loginProviderPreflightStatusSchema.nullable(),
  last_preflight_issues: z.array(z.string().min(1).max(255)),
  binding_count: z.number().int().nonnegative(),
  lock_version: z.number().int().nonnegative(),
  created_at: z.string().min(1),
  updated_at: z.string().min(1),
  callback_uri: z.url().nullable().optional(),
});
export const loginProviderConfigSchema = rawLoginProviderConfigSchema.transform((value): GeneratedLoginProviderConfig => {
  const { credential_fingerprint: credentialFingerprint, callback_uri: callbackURI, ...required } = value;
  return {
    ...required,
    ...(credentialFingerprint ? { credential_fingerprint: credentialFingerprint } : {}),
    ...(callbackURI ? { callback_uri: callbackURI } : {}),
  };
});
export type LoginProviderConfig = GeneratedLoginProviderConfig;

export const loginProviderConfigPageSchema = z.object({
  items: z.array(loginProviderConfigSchema),
  page: z.number().int().positive(),
  page_size: z.number().int().positive(),
  total: z.number().int().nonnegative(),
});
export type LoginProviderConfigPage = z.infer<typeof loginProviderConfigPageSchema>;

export const appLoginProviderBindingSchema = z.object({
  id: z.uuid().nullable(),
  app_id: z.uuid(),
  provider_code: loginProviderCodeSchema,
  login_provider_config_id: z.uuid().nullable(),
  config_name: z.string().max(160).nullable(),
  config_status: loginProviderConfigStatusSchema.nullable(),
  preflight_status: loginProviderPreflightStatusSchema.nullable(),
  enabled: z.boolean(),
  sort_order: z.number().int().nonnegative(),
  lock_version: z.number().int().nonnegative(),
  updated_at: z.string().nullable(),
});
export type AppLoginProviderBinding = GeneratedLoginProviderBinding;

export const appLoginProviderBindingsSchema = z.object({
  items: z.array(appLoginProviderBindingSchema),
});

export interface LoginProviderConfigFilters {
  q: string;
  provider_code?: LoginProviderCode;
  status?: LoginProviderConfigStatus;
  page: number;
  page_size: number;
}

export type LoginProviderFieldValue = string | string[];
export interface LoginProviderConfigFormValues {
  name: string;
  description: string;
  provider_code: LoginProviderCode;
  external_client_id: string;
  public_values: Record<string, LoginProviderFieldValue>;
}

export type LoginProviderConfigWriteInput = LoginProviderConfigRequest | LoginProviderConfigUpdateRequest;
export type LoginProviderSecretRotationInput = LoginProviderSecretRequestWritable;
export type AppLoginProviderBindingWriteItem = LoginProviderBindingInput;
export interface AppLoginProviderBindingsWriteInput {
  bindings: LoginProviderBindingBulkRequest["bindings"][number][];
}

const appLoginProviderBindingWriteItemSchema = z.object({
    provider_code: loginProviderCodeSchema,
    login_provider_config_id: z.uuid().nullable(),
    enabled: z.boolean(),
    sort_order: z.number().int().nonnegative(),
    lock_version: z.number().int().nonnegative(),
});

export const appLoginProviderBindingsWriteSchema = z.object({
  bindings: z.tuple([
    appLoginProviderBindingWriteItemSchema,
    appLoginProviderBindingWriteItemSchema,
    appLoginProviderBindingWriteItemSchema,
    appLoginProviderBindingWriteItemSchema,
  ]),
}).superRefine((value, ctx) => {
  const seen = new Set<LoginProviderCode>();
  for (const [index, item] of value.bindings.entries()) {
    if (seen.has(item.provider_code)) ctx.addIssue({ code: "custom", path: ["bindings", index, "provider_code"] });
    seen.add(item.provider_code);
    if (item.login_provider_config_id === null && item.enabled) {
      ctx.addIssue({ code: "custom", path: ["bindings", index, "enabled"] });
    }
  }
  if (seen.size !== loginProviderCodes.length) ctx.addIssue({ code: "custom", path: ["bindings"] });
});

export interface LoginProviderFieldDefinition {
  name: string;
  labelKey: string;
  helpKey: string;
  valueType: "string" | "url" | "string_array";
  required: boolean;
  maxLength: number;
  requiresRebuild: boolean;
}

export interface LoginProviderSecretDefinition {
  name: string;
  labelKey: string;
  helpKey: string;
  valueType: "string" | "pem";
  maxLength: number;
}

export interface LoginProviderDefinition {
  code: LoginProviderCode;
  externalClientLabelKey: string;
  externalClientHelpKey: string;
  platforms: readonly LoginProviderPlatform[];
  buildVariants: readonly (typeof loginProviderBuildVariants)[number][];
  authorizationKind: (typeof loginProviderAuthorizationKinds)[number];
  publicFields: readonly LoginProviderFieldDefinition[];
  secretFields: readonly LoginProviderSecretDefinition[];
  fromPublicConfig: (value: Record<string, unknown>) => Record<string, LoginProviderFieldValue>;
  toPublicConfig: (value: Record<string, LoginProviderFieldValue>) => LoginProviderPublicConfig;
}

function text(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function textArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}

function object(value: unknown): Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? value as Record<string, unknown> : {};
}

const field = (
  name: string,
  valueType: LoginProviderFieldDefinition["valueType"],
  required: boolean,
  requiresRebuild: boolean,
  maxLength = 512,
): LoginProviderFieldDefinition => ({
  name,
  labelKey: `login_providers.field.${name}`,
  helpKey: `login_providers.field_help.${name}`,
  valueType,
  required,
  maxLength,
  requiresRebuild,
});

const secret = (name: string, valueType: LoginProviderSecretDefinition["valueType"], maxLength = 65_536): LoginProviderSecretDefinition => ({
  name,
  labelKey: `login_providers.field.${name}`,
  helpKey: `login_providers.field_help.${name}`,
  valueType,
  maxLength,
});

const wechatDefinition: LoginProviderDefinition = {
  code: "wechat",
  externalClientLabelKey: "login_providers.field.wechat_app_id",
  externalClientHelpKey: "login_providers.field_help.wechat_app_id",
  platforms: ["android", "ios", "harmony"],
  buildVariants: ["android_google", "android_china", "ios", "harmony"],
  authorizationKind: "native_code",
  publicFields: [
    field("android_package_name", "string", false, true, 255),
    field("android_app_signature", "string", false, true, 256),
    field("ios_bundle_id", "string", false, true, 255),
    field("ios_universal_link", "url", false, true, 2048),
    field("harmony_bundle_name", "string", false, true, 255),
  ],
  secretFields: [secret("app_secret", "string", 4096)],
  fromPublicConfig: (value) => {
    const android = object(value["android"]);
    const ios = object(value["ios"]);
    const harmony = object(value["harmony"]);
    return {
      android_package_name: text(android["package_name"]),
      android_app_signature: text(android["app_signature"]),
      ios_bundle_id: text(ios["bundle_id"]),
      ios_universal_link: text(ios["universal_link"]),
      harmony_bundle_name: text(harmony["bundle_name"]),
    };
  },
  toPublicConfig: (value) => {
    const androidPackageName = text(value["android_package_name"]).trim();
    const androidAppSignature = text(value["android_app_signature"]).trim();
    const iosBundleId = text(value["ios_bundle_id"]).trim();
    const iosUniversalLink = text(value["ios_universal_link"]).trim();
    const harmonyBundleName = text(value["harmony_bundle_name"]).trim();
    return {
      android: { enabled: Boolean(androidPackageName && androidAppSignature), package_name: androidPackageName, app_signature: androidAppSignature },
      ios: { enabled: Boolean(iosBundleId && iosUniversalLink), bundle_id: iosBundleId, universal_link: iosUniversalLink },
      harmony: { enabled: Boolean(harmonyBundleName), bundle_name: harmonyBundleName },
    };
  },
};

const githubDefinition: LoginProviderDefinition = {
  code: "github",
  externalClientLabelKey: "login_providers.field.client_id",
  externalClientHelpKey: "login_providers.field_help.github_client_id",
  platforms: ["android", "ios", "harmony"],
  buildVariants: ["android_google", "android_china", "ios", "harmony"],
  authorizationKind: "browser_ticket",
  publicFields: [field("app_return_uri", "url", true, true, 2048)],
  secretFields: [secret("client_secret", "string", 4096)],
  fromPublicConfig: (value) => ({ app_return_uri: text(value["app_return_uri"]) }),
  toPublicConfig: (value) => ({ app_return_uri: text(value["app_return_uri"]).trim() }),
};

const appleDefinition: LoginProviderDefinition = {
  code: "apple",
  externalClientLabelKey: "login_providers.field.apple_client_id",
  externalClientHelpKey: "login_providers.field_help.apple_client_id",
  platforms: ["ios"],
  buildVariants: ["ios"],
  authorizationKind: "native_id_token",
  publicFields: [field("team_id", "string", true, false, 10), field("key_id", "string", true, false, 10)],
  secretFields: [secret("private_key_p8", "pem")],
  fromPublicConfig: (value) => ({ team_id: text(value["team_id"]), key_id: text(value["key_id"]) }),
  toPublicConfig: (value) => ({ team_id: text(value["team_id"]).trim(), key_id: text(value["key_id"]).trim() }),
};

const googleDefinition: LoginProviderDefinition = {
  code: "google",
  externalClientLabelKey: "login_providers.field.server_client_id",
  externalClientHelpKey: "login_providers.field_help.server_client_id",
  platforms: ["android"],
  buildVariants: ["android_google"],
  authorizationKind: "native_id_token",
  publicFields: [
    field("android_package_name", "string", true, true, 255),
    field("android_certificate_sha256", "string_array", true, true, 95),
  ],
  secretFields: [],
  fromPublicConfig: (value) => ({
    android_package_name: text(value["android_package_name"]),
    android_certificate_sha256: textArray(value["android_certificate_sha256"]),
  }),
  toPublicConfig: (value) => ({
    android_package_name: text(value["android_package_name"]).trim(),
    android_certificate_sha256: textArray(value["android_certificate_sha256"]).map((item) => item.trim().toUpperCase()).filter(Boolean),
  }),
};

export const loginProviderDefinitions: Readonly<Record<LoginProviderCode, LoginProviderDefinition>> = {
  wechat: wechatDefinition,
  github: githubDefinition,
  apple: appleDefinition,
  google: googleDefinition,
};

export function emptyLoginProviderConfigForm(provider: LoginProviderCode = "wechat"): LoginProviderConfigFormValues {
  const definition = loginProviderDefinitions[provider];
  return {
    name: "",
    description: "",
    provider_code: provider,
    external_client_id: "",
    public_values: Object.fromEntries(definition.publicFields.map((item) => [item.name, item.valueType === "string_array" ? [] : ""])),
  };
}

export function loginProviderConfigToForm(config: LoginProviderConfig): LoginProviderConfigFormValues {
  return {
    name: config.name,
    description: config.description,
    provider_code: config.provider_code,
    external_client_id: config.external_client_id,
    public_values: loginProviderDefinitions[config.provider_code].fromPublicConfig(config.public_config),
  };
}

export function mergeLoginProviderConflictValues(
  editableValues: LoginProviderConfigFormValues,
  latest: LoginProviderConfig,
): LoginProviderConfigFormValues {
  return { ...editableValues, provider_code: latest.provider_code };
}

export type LoginProviderConfigAction = "edit" | "rotate_secret" | "preflight" | "activate" | "disable" | "delete";

export function loginProviderConfigActions(
  config: LoginProviderConfig,
  permissions: ReadonlySet<string>,
): ReadonlySet<LoginProviderConfigAction> {
  const actions = new Set<LoginProviderConfigAction>();
  const definition = loginProviderDefinitions[config.provider_code];
  if (permissions.has("sys.login_provider_config.update")) {
    actions.add("edit");
    if (config.status === "active") actions.add("disable");
    else if (config.last_preflight_status === "ready") actions.add("activate");
  }
  if (definition.secretFields.length > 0 && permissions.has("sys.login_provider_config.rotate_secret")) actions.add("rotate_secret");
  if (permissions.has("sys.login_provider_config.preflight")) actions.add("preflight");
  if (permissions.has("sys.login_provider_config.delete")) actions.add("delete");
  return actions;
}

const androidPackagePattern = /^[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)+$/;
const appleIdentifierPattern = /^[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+$/;
const appleAccountIdentifierPattern = /^[A-Z0-9]{10}$/;
const certificateSha256Pattern = /^(?:[A-F0-9]{2}:){31}[A-F0-9]{2}$/;

export interface LoginProviderFormIssue {
  field: string;
  key: string;
}

export function validateLoginProviderConfigForm(values: LoginProviderConfigFormValues): LoginProviderFormIssue[] {
  const issues: LoginProviderFormIssue[] = [];
  const definition = loginProviderDefinitions[values.provider_code];
  if (!values.name.trim() || values.name.trim().length > 160) issues.push({ field: "name", key: "login_providers.validation.name" });
  if (values.description.length > 2000) issues.push({ field: "description", key: "login_providers.validation.description" });
  const external = values.external_client_id.trim();
  if (!external || external.length > 255) issues.push({ field: "external_client_id", key: "login_providers.validation.external_client_id" });
  if (values.provider_code === "wechat" && !/^wx[A-Za-z0-9]{16}$/.test(external)) issues.push({ field: "external_client_id", key: "login_providers.validation.wechat_app_id" });
  if (values.provider_code === "apple" && !appleIdentifierPattern.test(external)) issues.push({ field: "external_client_id", key: "login_providers.validation.apple_client_id" });
  if (values.provider_code === "google" && !external.endsWith(".apps.googleusercontent.com")) issues.push({ field: "external_client_id", key: "login_providers.validation.server_client_id" });

  for (const item of definition.publicFields) {
    const raw = values.public_values[item.name];
    const blank = Array.isArray(raw) ? raw.length === 0 : !text(raw).trim();
    if (item.required && blank) issues.push({ field: item.name, key: "login_providers.validation.required" });
    if (Array.isArray(raw)) {
      if (raw.some((entry) => entry.length > item.maxLength)) issues.push({ field: item.name, key: "login_providers.validation.max_length" });
    } else if (text(raw).length > item.maxLength) issues.push({ field: item.name, key: "login_providers.validation.max_length" });
    if (item.valueType === "url" && !blank) {
      try {
        if (new URL(text(raw)).protocol !== "https:") issues.push({ field: item.name, key: "login_providers.validation.https_url" });
      } catch {
        issues.push({ field: item.name, key: "login_providers.validation.https_url" });
      }
    }
  }

  const publicValues = values.public_values;
  const androidPackage = text(publicValues["android_package_name"]).trim();
  if (androidPackage && !androidPackagePattern.test(androidPackage)) issues.push({ field: "android_package_name", key: "login_providers.validation.android_package_name" });
  const iosBundleId = text(publicValues["ios_bundle_id"]).trim();
  if (iosBundleId && !appleIdentifierPattern.test(iosBundleId)) issues.push({ field: "ios_bundle_id", key: "login_providers.validation.ios_bundle_id" });
  if (values.provider_code === "wechat") {
    const androidSignature = text(publicValues["android_app_signature"]).trim();
    if (Boolean(androidPackage) !== Boolean(androidSignature)) issues.push({ field: androidPackage ? "android_app_signature" : "android_package_name", key: "login_providers.validation.wechat_android_pair" });
    const universalLink = text(publicValues["ios_universal_link"]).trim();
    if (Boolean(iosBundleId) !== Boolean(universalLink)) issues.push({ field: iosBundleId ? "ios_universal_link" : "ios_bundle_id", key: "login_providers.validation.wechat_ios_pair" });
  }
  if (values.provider_code === "apple") {
    for (const key of ["team_id", "key_id"] as const) {
      if (!appleAccountIdentifierPattern.test(text(publicValues[key]).trim())) issues.push({ field: key, key: `login_providers.validation.${key}` });
    }
  }
  if (values.provider_code === "google") {
    const fingerprints = textArray(publicValues["android_certificate_sha256"]);
    if (fingerprints.some((value) => !certificateSha256Pattern.test(value.trim().toUpperCase()))) {
      issues.push({ field: "android_certificate_sha256", key: "login_providers.validation.android_certificate_sha256" });
    }
  }
  return issues.filter((issue, index, all) => all.findIndex((candidate) => candidate.field === issue.field && candidate.key === issue.key) === index);
}

export function buildLoginProviderConfigInput(
  values: LoginProviderConfigFormValues,
  configSchemaVersion: number,
  lockVersion?: number,
): LoginProviderConfigWriteInput | null {
  if (configSchemaVersion !== 1 || validateLoginProviderConfigForm(values).length > 0) return null;
  const definition = loginProviderDefinitions[values.provider_code];
  return {
    name: values.name.trim(),
    description: values.description.trim(),
    provider_code: values.provider_code,
    external_client_id: values.external_client_id.trim(),
    config_schema_version: 1,
    public_config: definition.toPublicConfig(values.public_values),
    ...(lockVersion === undefined ? {} : { lock_version: lockVersion }),
  };
}

export function validateSecretValues(provider: LoginProviderCode, values: Record<string, string>): LoginProviderFormIssue[] {
  return loginProviderDefinitions[provider].secretFields.flatMap((fieldDefinition) => {
    const value = values[fieldDefinition.name]?.trim() ?? "";
    if (!value) return [{ field: fieldDefinition.name, key: "login_providers.validation.required" }];
    if (value.length > fieldDefinition.maxLength) return [{ field: fieldDefinition.name, key: "login_providers.validation.max_length" }];
    if (fieldDefinition.valueType === "pem" && (!value.includes("BEGIN PRIVATE KEY") || !value.includes("END PRIVATE KEY"))) {
      return [{ field: fieldDefinition.name, key: "login_providers.validation.private_key_p8" }];
    }
    return [];
  });
}

export function catalogMatchesDefinition(item: LoginProviderCatalogItem): boolean {
  const definition = loginProviderDefinitions[item.provider_code];
  const expected = [
    "external_client_id:external_client_id:false",
    ...definition.publicFields.map((fieldDefinition) => `${fieldDefinition.name}:public_config:false`),
    ...definition.secretFields.map((fieldDefinition) => `${fieldDefinition.name}:secret:true`),
  ].sort();
  const actual = item.fields.map((fieldDefinition) => `${fieldDefinition.name}:${fieldDefinition.location}:${String(fieldDefinition.secret)}`).sort();
  return item.authorization_kind === definition.authorizationKind
    && item.requires_secret === (definition.secretFields.length > 0)
    && JSON.stringify([...item.supported_platforms].sort()) === JSON.stringify([...definition.platforms].sort())
    && JSON.stringify([...item.build_variants].sort()) === JSON.stringify([...definition.buildVariants].sort())
    && JSON.stringify(actual) === JSON.stringify(expected);
}

export function readLoginProviderFilters(search: string): LoginProviderConfigFilters {
  const params = new URLSearchParams(search);
  const provider = loginProviderCodeSchema.safeParse(params.get("provider"));
  const status = loginProviderConfigStatusSchema.safeParse(params.get("status"));
  return {
    q: params.get("q") ?? "",
    ...(provider.success ? { provider_code: provider.data } : {}),
    ...(status.success ? { status: status.data } : {}),
    page: Math.max(1, Number(params.get("page")) || 1),
    page_size: 20,
  };
}

export const appLoginSettingsSchema = z.object({
  app_id: z.uuid(),
  password_enabled: z.literal(true),
  otp_enabled: z.boolean(),
  email_otp_enabled: z.boolean(),
  sms_otp_enabled: z.boolean(),
  oauth_enabled: z.boolean(),
  lock_version: z.number().int().nonnegative(),
  updated_at: z.string().min(1),
});
export type AppLoginSettings = z.infer<typeof appLoginSettingsSchema>;
export const appLoginSettingsInputSchema = z.object({
  otp_enabled: z.boolean(),
  email_otp_enabled: z.boolean(),
  sms_otp_enabled: z.boolean(),
  lock_version: z.number().int().nonnegative(),
}).refine((value) => !value.otp_enabled || value.email_otp_enabled || value.sms_otp_enabled, { path: ["otp_enabled"] });
export type AppLoginSettingsInput = z.infer<typeof appLoginSettingsInputSchema>;
