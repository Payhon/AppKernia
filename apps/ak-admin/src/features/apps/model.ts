import { z } from "zod";
import type { AdminApp, AdminAppPage, AdminAppUser } from "../../generated/api/types.gen";

export const appStatusSchema = z.enum(["active", "disabled"]);
export const registrationVerificationSchema = z.enum(["none", "email_otp"]);
export const appMembershipStatusSchema = z.enum(["pending_verification", "active", "disabled"]);

export type ManagedApplication = AdminApp;

export const applicationInputSchema = z.object({
  name: z.string().trim().min(1).max(160),
  default_locale: z.enum(["zh-CN", "en-US"]),
  registration_enabled: z.boolean(),
  registration_verification_mode: registrationVerificationSchema,
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

export type ManagedPage = AdminAppPage;

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
  const editable = (translation: { title: string; body_format: "markdown" | "blocks"; body: string | unknown[] }): AppPageEditorInput["translations"]["zh-CN"] => ({ title: translation.title, body_format: translation.body_format, body: typeof translation.body === "string" ? translation.body : JSON.stringify(translation.body, null, 2) });
  return { slug: page.slug, page_type: page.page_type, publish: "status" in page ? page.status === "published" : page.publish, translations: { "zh-CN": editable(page.translations["zh-CN"]), "en-US": editable(page.translations["en-US"]) }, lock_version: page.lock_version };
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
export interface AppListFilters { q: string; page: number; page_size: number; }
export interface MemberListFilters extends AppListFilters { status: string; }
export interface PageListFilters extends AppListFilters { q: string; status: string; }
