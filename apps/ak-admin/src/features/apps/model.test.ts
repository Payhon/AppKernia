import { describe, expect, it } from "vitest";
import { appMemberCreateInputSchema, appMemberPasswordResetSchema, appPageInputSchema, applicationInputSchema, getAppPageTitle, toAppPageEditorInput, toAppPageInput, type ManagedPage } from "./model";

describe("App management schemas", () => {
  it("requires a bounded application configuration", () => {
    const valid = {
      appid: "__UNI__MOBILE", app_type: "uni_app_x", name: "Mobile", description: "", introduction: "", remark: "",
      default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp",
      owner_type: "tenant", owner_id: "123e4567-e89b-12d3-a456-426614174000", icon_file_id: null,
      managers: [], members: [], screenshot_file_ids: [], channels: [], store_listings: [],
    };
    expect(applicationInputSchema.safeParse(valid).success).toBe(true);
    expect(applicationInputSchema.safeParse({ ...valid, appid: "not-a-manifest-id" }).success).toBe(false);
    expect(applicationInputSchema.safeParse({ name: "", default_locale: "fr-FR", registration_enabled: true, registration_verification_mode: "sms" }).success).toBe(false);
  });
  it("requires App user email and a minimum initial password", () => {
    expect(appMemberCreateInputSchema.safeParse({ email: "member@example.test", display_name: "Member", locale: "zh-CN", password: "correct-horse-battery" }).success).toBe(true);
    expect(appMemberCreateInputSchema.safeParse({ email: "invalid", display_name: "", locale: "zh-CN", password: "short" }).success).toBe(false);
  });
  it("requires password reset confirmation to match", () => {
    expect(appMemberPasswordResetSchema.safeParse({ new_password: "correct-horse-battery", confirm_password: "correct-horse-battery" }).success).toBe(true);
    expect(appMemberPasswordResetSchema.safeParse({ new_password: "correct-horse-battery", confirm_password: "different-password-value" }).success).toBe(false);
  });
  it("requires both legal-page locales and safe slugs", () => {
    expect(appPageInputSchema.safeParse({ slug: "privacy-policy", page_type: "privacy-policy", publish: true, translations: { "zh-CN": { title: "隐私政策", body_format: "markdown", body: "内容" }, "en-US": { title: "Privacy", body_format: "markdown", body: "Content" } } }).success).toBe(true);
    expect(appPageInputSchema.safeParse({ slug: "unsafe slug", page_type: "custom", publish: false, translations: { "zh-CN": { title: "", body_format: "markdown", body: "" }, "en-US": { title: "", body_format: "markdown", body: "" } } }).success).toBe(false);
  });
  it("preserves supported structured blocks through the editor JSON round trip", () => {
    const request = { slug: "about-us", page_type: "about-us" as const, publish: false, translations: { "zh-CN": { title: "关于", body_format: "blocks" as const, body: [{ type: "heading" as const, text: "标题", level: 2 }, { type: "paragraph" as const, text: "内容" }] }, "en-US": { title: "About", body_format: "markdown" as const, body: "# About" } } };
    const editor = toAppPageEditorInput(request);
    expect(editor.translations["zh-CN"].body).toContain("heading");
    expect(toAppPageInput(editor).translations["zh-CN"].body).toEqual(request.translations["zh-CN"].body);
    expect(appPageInputSchema.safeParse({ ...request, translations: { ...request.translations, "zh-CN": { ...request.translations["zh-CN"], body: [{ type: "unsupported" }] } } }).success).toBe(false);
  });
  it("keeps seeded reserved drafts editable and displayable before their first revision", () => {
    const seededDraft = { id: "page-id", slug: "privacy-policy", page_type: "privacy-policy", status: "draft", lock_version: 1, current_revision_id: null, updated_at: "2026-08-08T00:00:00Z", translations: {}, revisions: [] } as unknown as ManagedPage;

    expect(getAppPageTitle(seededDraft, "zh-CN")).toBeUndefined();
    expect(getAppPageTitle(seededDraft, "en-US")).toBeUndefined();
    expect(toAppPageEditorInput(seededDraft)).toMatchObject({
      slug: "privacy-policy",
      page_type: "privacy-policy",
      publish: false,
      translations: {
        "zh-CN": { title: "", body_format: "markdown", body: "" },
        "en-US": { title: "", body_format: "markdown", body: "" },
      },
    });
  });
  it("uses the alternate supported locale when a page title is untranslated", () => {
    const translatedPage = { id: "page-id", slug: "about-us", page_type: "about-us", status: "draft", lock_version: 1, updated_at: "2026-08-08T00:00:00Z", translations: { "zh-CN": { title: "关于我们", body_format: "markdown", body: "内容" } }, revisions: [] } as unknown as ManagedPage;

    expect(getAppPageTitle(translatedPage, "en-US")).toBe("关于我们");
  });
});
