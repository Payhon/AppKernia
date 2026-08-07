import { describe, expect, it } from "vitest";
import { appMemberCreateInputSchema, appMemberPasswordResetSchema, appPageInputSchema, applicationInputSchema, toAppPageEditorInput, toAppPageInput } from "./model";

describe("App management schemas", () => {
  it("requires a bounded application configuration", () => {
    expect(applicationInputSchema.safeParse({ name: "Mobile", default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp" }).success).toBe(true);
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
});
