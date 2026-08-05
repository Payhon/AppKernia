import { afterEach, describe, expect, it, vi } from "vitest";
import { authSession } from "../auth/store";
import { contentApi } from "./api";
import { articleInputSchema, categoryInputSchema } from "./model";

const categoryInput = {
  slug: "news",
  sort_order: 0,
  status: "active" as const,
  translations: {
    "zh-CN": { name: "新闻", description: "" },
    "en-US": { name: "News", description: "" },
  },
};

const categoryResponse = (lockVersion: number) => new Response(JSON.stringify({
  code: "OK",
  message: "OK",
  data: {
    id: "123e4567-e89b-12d3-a456-426614174000",
    ...categoryInput,
    lock_version: lockVersion,
    created_at: "2026-08-05T00:00:00Z",
    updated_at: "2026-08-05T00:00:00Z",
  },
  request_id: "content-test",
}), { status: 200, headers: { "Content-Type": "application/json" } });

const articleResponse = () => new Response(JSON.stringify({
  code: "OK",
  message: "OK",
  data: {
    id: "123e4567-e89b-12d3-a456-426614174000",
    category_id: "123e4567-e89b-12d3-a456-426614174000",
    slug: "welcome",
    status: "draft",
    featured: false,
    sort_order: 0,
    cover_file_id: null,
    cover_url: null,
    reading_minutes: 5,
    lock_version: 1,
    published_at: null,
    translations: {
      "zh-CN": { title: "欢迎", summary: "", body_format: "markdown", body: "正文" },
      "en-US": { title: "Welcome", summary: "", body_format: "blocks", body: [{ type: "paragraph", text: "Body" }] },
    },
    created_at: "2026-08-05T00:00:00Z",
    updated_at: "2026-08-05T00:00:00Z",
  },
  request_id: "content-test",
}), { status: 201, headers: { "Content-Type": "application/json" } });

const emptyListResponse = () => new Response(JSON.stringify({
  code: "OK",
  message: "OK",
  data: { items: [], page: 1, page_size: 20, total: 0 },
  request_id: "content-test",
}), { status: 200, headers: { "Content-Type": "application/json" } });

describe("content schemas", () => {
  afterEach(() => { vi.restoreAllMocks(); });

  it("requires both category translations and a slug", () => {
    expect(categoryInputSchema.safeParse({ slug: "news", sort_order: 0, status: "active", translations: { "zh-CN": { name: "新闻", description: "" }, "en-US": { name: "News", description: "" } } }).success).toBe(true);
    expect(categoryInputSchema.safeParse({ slug: "News", sort_order: 0, status: "active", translations: {} }).success).toBe(false);
  });

  it("accepts only contract article body formats and requires a JSON block array", () => {
    const valid = { slug: "welcome", category_id: "123e4567-e89b-12d3-a456-426614174000", cover_file_id: null, reading_minutes: 5, featured: false, sort_order: 0, translations: { "zh-CN": { title: "欢迎", summary: "", body_format: "markdown", body: "正文" }, "en-US": { title: "Welcome", summary: "", body_format: "blocks", body: "[{\"type\":\"paragraph\",\"text\":\"Body\"}]" } } };
    expect(articleInputSchema.safeParse(valid).success).toBe(true);
    expect(articleInputSchema.safeParse({ ...valid, translations: { ...valid.translations, "en-US": { ...valid.translations["en-US"], body_format: "html" } } }).success).toBe(false);
    expect(articleInputSchema.safeParse({ ...valid, translations: { ...valid.translations, "en-US": { ...valid.translations["en-US"], body: "not JSON" } } }).success).toBe(false);
  });

  it("serializes validated block editor text as the OpenAPI block array", async () => {
    const request = vi.spyOn(authSession, "adminRequest").mockResolvedValueOnce(articleResponse());
    await contentApi.createArticle({
      slug: "welcome",
      category_id: "123e4567-e89b-12d3-a456-426614174000",
      cover_file_id: null,
      reading_minutes: 5,
      featured: false,
      sort_order: 0,
      translations: {
        "zh-CN": { title: "欢迎", summary: "", body_format: "markdown", body: "正文" },
        "en-US": { title: "Welcome", summary: "", body_format: "blocks", body: "[{\"type\":\"paragraph\",\"text\":\"Body\"}]" },
      },
    });
    const body = JSON.parse(request.mock.calls[0]?.[1]?.body as string) as { translations: Record<string, { body: unknown }> };
    expect(body.translations["en-US"]?.body).toEqual([{ type: "paragraph", text: "Body" }]);
  });

  it("uses the final OpenAPI query parameter names and sort enums", async () => {
    const request = vi.spyOn(authSession, "adminRequest")
      .mockResolvedValueOnce(emptyListResponse())
      .mockResolvedValueOnce(emptyListResponse());
    await contentApi.articles({ q: "welcome", category_id: "123e4567-e89b-12d3-a456-426614174000", status: "published", page: 2, page_size: 50, sort: "published_desc" });
    await contentApi.categories({ q: "guides", status: "active", page: 3, page_size: 10, sort: "sort_order" });
    expect(request.mock.calls[0]?.[0]).toContain("category_id=123e4567-e89b-12d3-a456-426614174000");
    expect(request.mock.calls[0]?.[0]).toContain("page_size=50");
    expect(request.mock.calls[0]?.[0]).toContain("sort=published_desc");
    expect(request.mock.calls[0]?.[0]).not.toContain("category=");
    expect(request.mock.calls[0]?.[0]).not.toMatch(/[?&]size=/);
    expect(request.mock.calls[1]?.[0]).toContain("page_size=10");
  });

  it("maps category lock versions after create and update, then sends the version required for conflict detection", async () => {
    const request = vi.spyOn(authSession, "adminRequest")
      .mockResolvedValueOnce(categoryResponse(7))
      .mockResolvedValueOnce(categoryResponse(8));

    const created = await contentApi.createCategory(categoryInput);
    expect(created.version).toBe(7);

    const updated = await contentApi.updateCategory(created.id, { ...categoryInput, version: created.version });
    expect(updated.version).toBe(8);
    expect(request).toHaveBeenNthCalledWith(2, `/content/categories/${created.id}`, expect.objectContaining({ method: "PATCH" }));

    const updateBody = request.mock.calls[1]?.[1]?.body;
    expect(typeof updateBody).toBe("string");
    expect(JSON.parse(updateBody as string)).toMatchObject({ lock_version: 7 });
  });
});
