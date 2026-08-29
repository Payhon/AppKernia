import { afterEach, describe, expect, it, vi } from "vitest";
import { authSession } from "../auth/store";
import { contentApi } from "./api";
import { articleInputSchema, articlePublishSchema, blocksToMarkdown, categoryInputSchema } from "./model";

const base = { content_type: "article" as const, slug: "welcome", category_ids: ["123e4567-e89b-12d3-a456-426614174000"], topic_id: null, tag_ids: [], media: [], cover_file_id: null, reading_minutes: 5, allow_comments: true, pinned: false, featured: false, latest: false, video_source_type: null, video_file_id: null, video_external_url: null, video_duration_seconds: null, sort_order: 0, translations: { "zh-CN": { title: "欢迎", summary: "摘要", body_format: "markdown" as const, body: "# 欢迎\n\n正文" }, "en-US": { title: "Welcome", summary: "Summary", body_format: "markdown" as const, body: "# Welcome\n\nBody" } } };

describe("content schemas", () => {
  afterEach(() => vi.restoreAllMocks());
  it("requires both category translations and a slug", () => {
    expect(categoryInputSchema.safeParse({ parent_id: null, image_file_id: null, slug: "news", sort_order: 0, status: "active", translations: { "zh-CN": { name: "新闻", description: "" }, "en-US": { name: "News", description: "" } } }).success).toBe(true);
    expect(categoryInputSchema.safeParse({ slug: "News" }).success).toBe(false);
  });
  it("accepts Markdown and rejects unsafe protocols or HTML", () => {
    expect(articleInputSchema.safeParse(base).success).toBe(true);
    expect(articleInputSchema.safeParse({ ...base, translations: { ...base.translations, "en-US": { ...base.translations["en-US"], body: "![x](javascript:alert(1))" } } }).success).toBe(false);
    expect(articleInputSchema.safeParse({ ...base, translations: { ...base.translations, "en-US": { ...base.translations["en-US"], body: "<script>alert(1)</script>" } } }).success).toBe(false);
  });
  it("allows incomplete bilingual drafts but rejects them at publish validation", () => {
    const draft = { ...base, category_ids: [], translations: { "zh-CN": { title: "草稿", summary: "", body_format: "markdown" as const, body: "" }, "en-US": { title: "", summary: "", body_format: "markdown" as const, body: "" } } };
    expect(articleInputSchema.safeParse(draft).success).toBe(true);
    expect(articlePublishSchema.safeParse(draft).success).toBe(false);
  });
  it("converts legacy blocks deterministically", () => {
    expect(blocksToMarkdown({ type: "doc", content: [{ type: "heading", attrs: { level: 2 }, content: [{ type: "text", text: "Title" }] }, { type: "paragraph", content: [{ type: "text", text: "Body" }] }] })).toBe("## Title\n\nBody");
  });
  it("sends Markdown strings to the API adapter", async () => {
    const response = new Response(JSON.stringify({ code: "OK", message: "OK", data: { ...base, id: "123e4567-e89b-12d3-a456-426614174000", category_id: base.category_ids[0], tags: [], media: [], status: "draft", lock_version: 1, published_at: null, updated_at: "2026-08-05T00:00:00Z", created_at: "2026-08-05T00:00:00Z" }, request_id: "content-test" }), { status: 201, headers: { "Content-Type": "application/json" } });
    const request = vi.spyOn(authSession, "adminRequest").mockResolvedValueOnce(response);
    await contentApi.createArticle(base);
    const payload = JSON.parse(request.mock.calls[0]?.[1]?.body as string) as { translations: Record<string, { body: unknown; body_format: string }> };
    expect(payload.translations["en-US"]?.body).toBe("# Welcome\n\nBody");
    expect(payload.translations["en-US"]?.body_format).toBe("markdown");
  });
  it("maps the UI version to lock_version without leaking an unknown version field", async () => {
    const response = new Response(JSON.stringify({ code: "OK", message: "OK", data: { ...base, id: "123e4567-e89b-12d3-a456-426614174000", category_id: base.category_ids[0], tags: [], media: [], status: "draft", lock_version: 4, published_at: null, updated_at: "2026-08-05T00:00:00Z", created_at: "2026-08-05T00:00:00Z" }, request_id: "content-update-test" }), { status: 200, headers: { "Content-Type": "application/json" } });
    const request = vi.spyOn(authSession, "adminRequest").mockResolvedValueOnce(response);

    await contentApi.updateArticle("123e4567-e89b-12d3-a456-426614174000", { ...base, version: 3 });

    const payload = JSON.parse(request.mock.calls[0]?.[1]?.body as string) as Record<string, unknown>;
    expect(payload["lock_version"]).toBe(3);
    expect(payload).not.toHaveProperty("version");
  });
});
