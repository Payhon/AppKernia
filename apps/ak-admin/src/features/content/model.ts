import { z } from "zod";

export const contentStatusSchema = z.enum(["draft", "published", "archived"]);
export const contentTypeSchema = z.enum(["article", "gallery", "video"]);
export type ContentStatus = z.infer<typeof contentStatusSchema>;
export type ContentType = z.infer<typeof contentTypeSchema>;
export type Localized<T> = Record<"zh-CN" | "en-US", T>;

export interface ContentCategory { id: string; parent_id: string | null; image_file_id: string | null; image_url: string | null; slug: string; sort_order: number; status: "active" | "disabled"; version: number; updated_at: string; translations: Localized<{ name: string; description: string }>; }
export interface ContentTopic { id: string; slug: string; cover_file_id: string | null; cover_url: string | null; sort_order: number; status: "active" | "disabled"; version: number; updated_at: string; translations: Localized<{ name: string; description: string }>; }
export interface ContentTag { id: string; name: string; status: "active" | "disabled"; version: number; usage_count: number; }
export interface ContentMedia { id: string; file_id: string; role: "gallery" | "inline"; sort_order: number; translations: Localized<{ alt_text: string }>; }
export interface ContentComment { id: string; article_id: string; author_id: string; author_name: string; author_avatar_url: string | null; parent_id: string | null; root_id: string | null; status: "pending" | "approved" | "rejected" | "hidden" | "deleted"; body: string; moderation_reason: string | null; created_at: string; updated_at: string; }
export interface ContentCommentReport { id: string; comment_id: string; reporter_id: string; reason: "spam" | "abuse" | "illegal" | "privacy" | "other"; details: string; status: "open" | "resolved" | "dismissed"; resolution: string; created_at: string; resolved_at: string | null; }
export interface ContentArticle {
  share_url?: string;
  id: string; slug: string; content_type: ContentType; category_id: string; category_ids: string[]; topic_id: string | null; tag_ids: string[]; tags: ContentTag[]; media: ContentMedia[];
  cover_file_id: string | null; cover_url: string | null; reading_minutes: number; allow_comments: boolean; pinned: boolean; featured: boolean; latest: boolean;
  video_source_type: "upload" | "external" | null; video_file_id: string | null; video_external_url: string | null; video_duration_seconds: number | null;
  sort_order: number; status: ContentStatus; published_at: string | null; version: number; updated_at: string;
  translations: Localized<{ title: string; summary: string; body_format: "markdown"; body: string }>;
}
export interface ContentList<T> { items: T[]; total: number; page?: number; page_size?: number; }

const categoryTranslation = z.object({ name: z.string().trim().min(1).max(160), description: z.string().trim().max(500) });
export const categoryInputSchema = z.object({ parent_id: z.uuid().nullable(), image_file_id: z.uuid().nullable(), slug: z.string().trim().min(1).max(160).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/), translations: z.object({ "zh-CN": categoryTranslation, "en-US": categoryTranslation }), sort_order: z.number().int().min(0).max(99999), status: z.enum(["active", "disabled"]), version: z.number().int().positive().optional() });
export type CategoryInput = z.infer<typeof categoryInputSchema>;

const topicTranslation = z.object({ name: z.string().trim().min(1).max(160), description: z.string().trim().max(2000) });
export const topicInputSchema = z.object({ slug: z.string().trim().min(1).max(160).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/), cover_file_id: z.uuid().nullable(), translations: z.object({ "zh-CN": topicTranslation, "en-US": topicTranslation }), sort_order: z.number().int().min(0).max(99999), status: z.enum(["active", "disabled"]), version: z.number().int().positive().optional() });
export type TopicInput = z.infer<typeof topicInputSchema>;

const mediaSchema = z.object({ id: z.uuid(), file_id: z.uuid(), role: z.enum(["gallery", "inline"]), sort_order: z.number().int().min(0), translations: z.object({ "zh-CN": z.object({ alt_text: z.string().trim().max(500) }), "en-US": z.object({ alt_text: z.string().trim().max(500) }) }) });
const markdownBodySchema = z.string().max(100_000).superRefine((value, context) => {
  if (/<\/?[a-z][^>]*>/i.test(value) || /(?:javascript|data|vbscript):/i.test(value)) {
    context.addIssue({ code: "custom", message: "unsafe markdown" });
  }
  const mediaLinks = value.match(/!?\[[^\]]*\]\(([^)\s]+)(?:\s+[^)]*)?\)/g) ?? [];
  for (const link of mediaLinks) {
    const target = link.replace(/^!?\[[^\]]*\]\(/, "").split(/[\s)]/, 1)[0] ?? "";
    if (target.startsWith("/api/v1/public/content/assets/")) continue;
    try {
      if (new URL(target).protocol !== "https:") context.addIssue({ code: "custom", message: "external media must use HTTPS" });
    } catch {
      context.addIssue({ code: "custom", message: "invalid markdown URL" });
    }
  }
});
const articleTranslation = z.object({ title: z.string().trim().max(300), summary: z.string().trim().max(3000), body_format: z.literal("markdown"), body: markdownBodySchema });
const articleInputBaseSchema = z.object({
  content_type: contentTypeSchema, slug: z.string().trim().min(1).max(200).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/), category_ids: z.array(z.uuid()).max(10), topic_id: z.uuid().nullable(), tag_ids: z.array(z.uuid()).max(10), media: z.array(mediaSchema).max(9),
  cover_file_id: z.uuid().nullable(), reading_minutes: z.number().int().min(1).max(120), allow_comments: z.boolean(), pinned: z.boolean(), featured: z.boolean(), latest: z.boolean(), sort_order: z.number().int().min(0).max(99999),
  video_source_type: z.enum(["upload", "external"]).nullable(), video_file_id: z.uuid().nullable(), video_external_url: z.url().nullable().superRefine((value, context) => { if (value) { try { const parsed = new URL(value); if (parsed.protocol !== "https:" || parsed.username || parsed.password || !/\.(?:mp4|m3u8)(?:$|[?#])/i.test(parsed.pathname)) context.addIssue({ code: "custom", message: "video URL must be a direct HTTPS MP4 or HLS URL" }); } catch { context.addIssue({ code: "custom", message: "invalid video URL" }); } } }), video_duration_seconds: z.number().int().min(0).nullable(), translations: z.object({ "zh-CN": articleTranslation, "en-US": articleTranslation }), version: z.number().int().positive().optional(),
});
function refineArticleInput(value: z.infer<typeof articleInputBaseSchema>, context: z.RefinementCtx, publish: boolean) {
  for (const locale of ["zh-CN", "en-US"] as const) {
    const max = value.content_type === "article" ? 30_000 : value.content_type === "gallery" ? 3_000 : 1_000;
    const length = value.content_type === "article" ? Array.from(value.translations[locale].body).length : Array.from(value.translations[locale].summary).length;
    if (length > max) context.addIssue({ code: "custom", path: ["translations", locale, value.content_type === "article" ? "body" : "summary"], message: "content too long" });
    const summaryLimit = value.content_type === "gallery" ? 3_000 : 1_000;
    if (Array.from(value.translations[locale].summary).length > summaryLimit) context.addIssue({ code: "custom", path: ["translations", locale, "summary"], message: "summary too long" });
    if (publish && value.translations[locale].title.trim().length === 0) context.addIssue({ code: "custom", path: ["translations", locale, "title"], message: "title required for publishing" });
    if (publish && value.translations[locale].summary.trim().length === 0) context.addIssue({ code: "custom", path: ["translations", locale, "summary"], message: "summary required for publishing" });
    if (publish && value.content_type === "article" && length === 0) context.addIssue({ code: "custom", path: ["translations", locale, "body"], message: "body required for publishing" });
  }
  if (publish && value.category_ids.length < 1) context.addIssue({ code: "custom", path: ["category_ids"], message: "category required for publishing" });
  if (value.content_type === "gallery" && ((publish && value.media.length < 1) || value.media.length > 9)) context.addIssue({ code: "custom", path: ["media"], message: "gallery requires 1-9 images" });
  if (publish && value.media.some((item) => item.translations["zh-CN"].alt_text.trim().length === 0 || item.translations["en-US"].alt_text.trim().length === 0)) context.addIssue({ code: "custom", path: ["media"], message: "media alt text required for publishing" });
  if (value.content_type === "video" && value.video_source_type === "upload" && !value.video_file_id) context.addIssue({ code: "custom", path: ["video_file_id"], message: "video file required" });
  if (value.content_type === "video" && value.video_source_type === "external" && !value.video_external_url) context.addIssue({ code: "custom", path: ["video_external_url"], message: "video URL required" });
}

export function blocksToMarkdown(value: unknown): string {
  const node = (input: unknown): string => {
    if (typeof input === "string") return input;
    if (!input || typeof input !== "object") return "";
    const item = input as { type?: string; text?: string; attrs?: Record<string, unknown>; content?: unknown[]; marks?: { type?: string; attrs?: Record<string, unknown> }[] };
    if (item.type === "text") {
      let text = item.text ?? "";
      for (const mark of item.marks ?? []) {
        if (mark.type === "bold") text = `**${text}**`;
        if (mark.type === "italic") text = `*${text}*`;
        if (mark.type === "code") text = `\`${text}\``;
        if (mark.type === "link" && typeof mark.attrs?.["href"] === "string") text = `[${text}](${mark.attrs["href"]})`;
      }
      return text;
    }
    if (item.type === "image" && typeof item.attrs?.["file_id"] === "string") { const alt = typeof item.attrs["alt"] === "string" ? item.attrs["alt"] : ""; return `![${alt}](/api/v1/public/content/assets/${item.attrs["file_id"]})`; }
    const children = (item.content ?? []).map(node).join("");
    switch (item.type) {
      case "heading": return `${"#".repeat(Math.min(6, Math.max(1, Number(item.attrs?.["level"] ?? 2))))} ${children}\n\n`;
      case "paragraph": return `${children}\n\n`;
      case "blockquote": return children.split("\n").filter(Boolean).map((line) => `> ${line}`).join("\n") + "\n\n";
      case "bulletList": return (item.content ?? []).map((child) => `- ${node(child).trim()}`).join("\n") + "\n\n";
      case "orderedList": return (item.content ?? []).map((child, index) => `${String(index + 1)}. ${node(child).trim()}`).join("\n") + "\n\n";
      case "listItem": return children;
      case "codeBlock": return `\`\`\`\n${children}\n\`\`\`\n\n`;
      case "horizontalRule": return "---\n\n";
      case "hardBreak": return "\n";
      default: return children;
    }
  };
  if (typeof value === "string") return value;
  if (Array.isArray(value)) return value.map(node).join("").trim();
  return node(value).trim();
}

export function normalizeArticleBody(body: unknown, format?: string): string {
  if (format === "markdown" && typeof body === "string") return body;
  return blocksToMarkdown(body);
}
export const articleInputSchema = articleInputBaseSchema.superRefine((value, context) => { refineArticleInput(value, context, false); });
export const articlePublishSchema = articleInputBaseSchema.superRefine((value, context) => { refineArticleInput(value, context, true); });
export type ArticleInput = z.infer<typeof articleInputSchema>;

export interface ArticleFilters { q: string; category_id: string; topic_id: string; type: string; status: string; featured: string; page: number; page_size: number; sort: "updated_desc" | "published_desc" | "sort_order" | "slug"; }
export interface CategoryFilters { q: string; status: string; page: number; page_size: number; sort: "sort_order" | "updated_desc" | "slug"; }
export interface TopicFilters { q: string; status: string; page: number; page_size: number; }
export type TagFilters = TopicFilters;
export interface CommentFilters { article_id: string; status: string; page: number; page_size: number; }
export interface CommentReportFilters { status: string; page: number; page_size: number; }
