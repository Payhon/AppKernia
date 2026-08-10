import { z } from "zod";
import type { AdminApp, AdminAppPage, AdminAppUser } from "../../generated/api/types.gen";

export const appStatusSchema = z.enum(["active", "disabled"]);
export const appTypeSchema = z.enum(["uni_app", "uni_app_x"]);
export const registrationVerificationSchema = z.enum(["none", "email_otp"]);
export const appMembershipStatusSchema = z.enum(["pending_verification", "active", "disabled"]);

export type ManagedApplication = AdminApp;

const optionalUuid = z.union([z.literal(""), z.uuid()]).transform((value) => value || null);
const optionalHttps = z.union([z.literal(""), z.url().refine((value) => value.startsWith("https://"))]).transform((value) => value || null);
export const applicationChannelCodes = ["android", "ios", "harmony", "h5", "quickapp", "mp_weixin", "mp_alipay", "mp_baidu", "mp_toutiao", "mp_qq", "mp_kuaishou", "mp_lark", "mp_jd", "mp_dingtalk"] as const;
export const applicationChannelSchema = z.object({
  id: z.uuid().optional(),
  channel_code: z.enum(applicationChannelCodes),
  name: z.string().trim().max(160),
  url: optionalHttps.nullable().optional(),
  abm_url: optionalHttps.nullable().optional(),
  qrcode_file_id: optionalUuid.nullable().optional(),
  enabled: z.boolean(),
});
export const applicationStoreListingSchema = z.object({
  id: z.uuid().optional(),
  name: z.string().trim().min(1).max(160),
  scheme: z.string().trim().max(255).refine((value) => value === "" || /^[A-Za-z][A-Za-z0-9+.-]*:\/\//.test(value)),
  enabled: z.boolean(),
  priority: z.number().int().min(-100000).max(100000),
});
export const applicationInputSchema = z.object({
  appid: z.string().trim().regex(/^__UNI__[A-Za-z0-9_]{2,120}$/),
  app_type: appTypeSchema,
  code: z.string().trim().regex(/^[a-z][a-z0-9-]{1,62}$/).optional(),
  name: z.string().trim().min(1).max(120),
  description: z.string().trim().max(4000),
  introduction: z.string().trim().max(1000),
  remark: z.string().trim().max(4000),
  default_locale: z.enum(["zh-CN", "en-US"]),
  registration_enabled: z.boolean(),
  registration_verification_mode: registrationVerificationSchema,
  owner_type: z.enum(["user", "tenant"]),
  owner_id: z.uuid(),
  icon_file_id: z.uuid().nullable(),
  managers: z.array(z.uuid()).max(100),
  members: z.array(z.uuid()).max(100),
  screenshot_file_ids: z.array(z.uuid()).max(20),
  channels: z.array(applicationChannelSchema).max(20),
  store_listings: z.array(applicationStoreListingSchema).max(100),
  lock_version: z.number().int().positive().optional(),
});
export type ApplicationInput = z.infer<typeof applicationInputSchema>;

export type AppMember = AdminAppUser;

export const appMemberCreateInputSchema = z.object({
  email: z.email().trim().max(320),
  display_name: z.string().trim().min(1).max(160),
  password: z.string().min(12).max(128).optional(),
  locale: z.enum(["zh-CN", "en-US"]),
});
export type AppMemberCreateInput = z.infer<typeof appMemberCreateInputSchema>;
export const appMemberUpdateInputSchema = z.object({ display_name: z.string().trim().min(1).max(160), lock_version: z.number().int().positive() });
export type AppMemberUpdateInput = z.infer<typeof appMemberUpdateInputSchema>;
export const appMemberPasswordResetSchema = z.object({
  new_password: z.string().min(12).max(128),
  confirm_password: z.string().min(12).max(128),
}).refine((value) => value.new_password === value.confirm_password, { path: ["confirm_password"] });
export type AppMemberPasswordResetInput = z.infer<typeof appMemberPasswordResetSchema>;

export type AppPageLocale = "zh-CN" | "en-US";
type PartialAppPageTranslation = Partial<AdminAppPage["translations"]["zh-CN"]>;
/** Reserved drafts are seeded before their first revision, so the API can legitimately omit translations. */
export type ManagedPage = Omit<AdminAppPage, "translations"> & { translations?: Partial<Record<AppPageLocale, PartialAppPageTranslation>> | null };

export const appPageBlockSchema = z.discriminatedUnion("type", [
  z.object({ type: z.literal("heading"), text: z.string().trim().min(1).max(10_000), level: z.number().int().min(1).max(6) }),
  z.object({ type: z.literal("paragraph"), text: z.string().trim().min(1).max(20_000) }),
  z.object({ type: z.literal("quote"), text: z.string().trim().min(1).max(20_000) }),
  z.object({ type: z.literal("code"), text: z.string().min(1).max(20_000), language: z.string().trim().min(1).max(64).optional() }),
  z.object({ type: z.literal("image"), url: z.url().max(2_000), alt: z.string().trim().max(300).optional() }),
  z.object({ type: z.literal("list"), items: z.array(z.string().trim().min(1).max(10_000)).min(1).max(100), ordered: z.boolean().optional() }),
  z.object({ type: z.literal("divider") }),
]);
export const appPageBlocksSchema = z.array(appPageBlockSchema).min(1).max(500);
export const appPageTranslationSchema = z.object({
  title: z.string().trim().min(1).max(300),
  body_format: z.enum(["markdown", "blocks"]),
  body: z.union([z.string().trim().min(1).max(100_000), appPageBlocksSchema]),
}).superRefine((value, context) => {
  if (value.body_format === "markdown" && typeof value.body !== "string") context.addIssue({ code: "custom", path: ["body"], message: "markdown body must be text" });
  if (value.body_format === "blocks" && !Array.isArray(value.body)) context.addIssue({ code: "custom", path: ["body"], message: "blocks body must be an array" });
});

export const appPageInputSchema = z.object({
  slug: z.string().trim().min(1).max(160).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
  page_type: z.enum(["privacy-policy", "terms-of-service", "about-us", "custom"]),
  publish: z.boolean(),
  translations: z.object({ "zh-CN": appPageTranslationSchema, "en-US": appPageTranslationSchema }),
  lock_version: z.number().int().positive().optional(),
});
export type AppPageInput = z.infer<typeof appPageInputSchema>;

/** The UI keeps blocks in a text area as formatted JSON, while requests preserve the array. */
export type AppPageEditorInput = Omit<AppPageInput, "translations"> & { translations: { "zh-CN": Omit<AppPageInput["translations"]["zh-CN"], "body"> & { body: string }; "en-US": Omit<AppPageInput["translations"]["en-US"], "body"> & { body: string } } };
export const appPageEditorInputSchema = z.object({
  slug: z.string().trim().min(1).max(160).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
  page_type: z.enum(["privacy-policy", "terms-of-service", "about-us", "custom"]),
  publish: z.boolean(),
  translations: z.object({
    "zh-CN": z.object({ title: z.string().trim().min(1).max(300), body_format: z.enum(["markdown", "blocks"]), body: z.string().trim().min(1).max(100_000) }),
    "en-US": z.object({ title: z.string().trim().min(1).max(300), body_format: z.enum(["markdown", "blocks"]), body: z.string().trim().min(1).max(100_000) }),
  }),
  lock_version: z.number().int().positive().optional(),
}).superRefine((value, context) => {
  (["zh-CN", "en-US"] as const).forEach((locale) => {
    if (value.translations[locale].body_format !== "blocks") return;
    try { appPageBlocksSchema.parse(JSON.parse(value.translations[locale].body)); }
    catch { context.addIssue({ code: "custom", path: ["translations", locale, "body"], message: "blocks body must be valid supported block JSON" }); }
  });
});
export function toAppPageEditorInput(page: AppPageInput | ManagedPage): AppPageEditorInput {
  const editable = (translation: PartialAppPageTranslation | undefined): AppPageEditorInput["translations"]["zh-CN"] => ({
    title: translation?.title ?? "",
    body_format: translation?.body_format ?? "markdown",
    body: typeof translation?.body === "string" ? translation.body : translation?.body ? JSON.stringify(translation.body, null, 2) : "",
  });
  return { slug: page.slug, page_type: page.page_type, publish: "status" in page ? page.status === "published" : page.publish, translations: { "zh-CN": editable(page.translations?.["zh-CN"]), "en-US": editable(page.translations?.["en-US"]) }, lock_version: page.lock_version };
}

/** Resolves a title for display while older reserved drafts have no revision or translations yet. */
export function getAppPageTitle(page: ManagedPage, locale: AppPageLocale): string | undefined {
  const alternateLocale: AppPageLocale = locale === "zh-CN" ? "en-US" : "zh-CN";
  return [page.translations?.[locale]?.title, page.translations?.[alternateLocale]?.title]
    .map((title) => title?.trim())
    .find((title): title is string => Boolean(title));
}
export function toAppPageInput(editor: AppPageEditorInput): AppPageInput {
  const parsed = appPageEditorInputSchema.parse(editor);
  const request = {
    ...parsed,
    translations: Object.fromEntries((["zh-CN", "en-US"] as const).map((locale) => {
      const translation = parsed.translations[locale];
      const blocks: unknown = translation.body_format === "blocks" ? JSON.parse(translation.body) : translation.body;
      return [locale, { ...translation, body: translation.body_format === "blocks" ? appPageBlocksSchema.parse(blocks) : translation.body }];
    })) as AppPageInput["translations"],
  };
  return appPageInputSchema.parse(request);
}

export interface Paginated<T> { items: T[]; total: number; }
export interface AppListFilters { q: string; status?: string; app_type?: string; page: number; page_size: number; }
export interface MemberListFilters extends AppListFilters { status: string; }
export interface PageListFilters extends AppListFilters { q: string; status: string; }
