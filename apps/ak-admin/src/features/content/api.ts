import { authSession } from "../auth/store";
import { toApiError } from "../../shared/api/error";
import type { AdminContentArticle, AdminContentArticleRequest, AdminContentCategory, AdminContentCategoryRequest, AdminContentTag, AdminContentTopic, ContentComment, ContentCommentReport } from "../../generated/api/types.gen";
import { normalizeArticleBody } from "./model";
import type { ArticleFilters, ArticleInput, CategoryFilters, CategoryInput, CommentFilters, CommentReportFilters, ContentArticle, ContentCategory, ContentComment as UiComment, ContentCommentReport as UiReport, ContentList, ContentTag, ContentTopic, TagFilters, TopicFilters, TopicInput } from "./model";

async function data<T>(path: `/${string}`, init?: RequestInit): Promise<T> { const response = await authSession.adminRequest(path, init); if (!response.ok) throw await toApiError(response); return ((await response.json()) as { data: T }).data; }
function query(filters: object) { const params = new URLSearchParams(); for (const [key, value] of Object.entries(filters)) if ((typeof value === "string" || typeof value === "number" || typeof value === "boolean") && value !== "") params.set(key, String(value)); return params.size ? `?${params}` : ""; }
const body = (method: "POST" | "PATCH", value: unknown): RequestInit => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(value) });
const contentPath = (appId: string | null | undefined, suffix: string) => (appId ? `/apps/${encodeURIComponent(appId)}/content${suffix}` : `/content${suffix}`) as `/${string}`;
const categoryRequest = (input: CategoryInput): AdminContentCategoryRequest => ({ parent_id: input.parent_id, image_file_id: input.image_file_id, slug: input.slug, sort_order: input.sort_order, status: input.status, translations: input.translations, ...(input.version ? { lock_version: input.version } : {}) });
const articleRequest = (input: ArticleInput): AdminContentArticleRequest => {
  const { version, ...request } = input;
  return { ...request, translations: Object.fromEntries(Object.entries(input.translations).map(([locale, translation]) => [locale, { ...translation, body_format: "markdown", body: translation.body }])) as unknown as AdminContentArticleRequest["translations"], ...(version ? { lock_version: version } : {}) };
};
const category = (value: AdminContentCategory): ContentCategory => ({ ...value, parent_id: value.parent_id ?? null, image_file_id: value.image_file_id ?? null, image_url: value.image_url ?? null, updated_at: value.updated_at, version: value.lock_version });
const topic = (value: AdminContentTopic): ContentTopic => ({ ...value, cover_file_id: value.cover_file_id ?? null, cover_url: value.cover_url ?? null, updated_at: value.updated_at, version: value.lock_version });
const tag = (value: AdminContentTag): ContentTag => ({ id: value.id, name: value.name, status: value.status ?? "active", usage_count: value.usage_count ?? 0, version: value.lock_version ?? 1 });
const comment = (value: ContentComment): UiComment => ({ ...value, author_avatar_url: value.author_avatar_url ?? null, parent_id: value.parent_id ?? null, root_id: value.root_id ?? null, moderation_reason: value.moderation_reason ?? null });
const report = (value: ContentCommentReport): UiReport => ({ ...value, resolved_at: value.resolved_at ?? null });
const article = (value: AdminContentArticle): ContentArticle => ({
  ...value, content_type: value.content_type, category_id: value.category_ids[0] ?? "", category_ids: value.category_ids, topic_id: value.topic_id ?? null, tag_ids: value.tag_ids, tags: value.tags.map(tag), media: value.media,
  cover_file_id: value.cover_file_id ?? null, cover_url: value.cover_url ?? null, published_at: value.published_at ?? null, video_source_type: value.video_source_type ?? null, video_file_id: value.video_file_id ?? null, video_external_url: value.video_external_url ?? null, video_duration_seconds: value.video_duration_seconds ?? null,
  translations: Object.fromEntries(Object.entries(value.translations).map(([locale, translation]) => [locale, { ...translation, body_format: "markdown", body: normalizeArticleBody(translation.body, translation.body_format) }])) as ContentArticle["translations"], version: value.lock_version,
});

export const contentApi = {
  categories: async (filters: CategoryFilters, appId?: string | null) => { const value = await data<ContentList<AdminContentCategory>>(contentPath(appId, `/categories${query(filters)}`)); return { ...value, items: value.items.map(category) }; },
  createCategory: async (input: CategoryInput, appId?: string | null) => category(await data<AdminContentCategory>(contentPath(appId, "/categories"), body("POST", categoryRequest(input)))),
  updateCategory: async (id: string, input: CategoryInput, appId?: string | null) => category(await data<AdminContentCategory>(contentPath(appId, `/categories/${encodeURIComponent(id)}`), body("PATCH", categoryRequest(input)))),
  deleteCategory: (id: string, version: number, appId?: string | null) => data<undefined>(contentPath(appId, `/categories/${encodeURIComponent(id)}${query({ lock_version: version })}`), { method: "DELETE" }),
  articles: async (filters: ArticleFilters, appId?: string | null) => { const value = await data<ContentList<AdminContentArticle>>(contentPath(appId, `/items${query(filters)}`)); return { ...value, items: value.items.map(article) }; },
  createArticle: async (input: ArticleInput, appId?: string | null) => article(await data<AdminContentArticle>(contentPath(appId, "/items"), body("POST", articleRequest(input)))),
  updateArticle: async (id: string, input: ArticleInput, appId?: string | null) => article(await data<AdminContentArticle>(contentPath(appId, `/items/${encodeURIComponent(id)}`), body("PATCH", articleRequest(input)))),
  deleteArticle: (id: string, version: number, appId?: string | null) => data<undefined>(contentPath(appId, `/items/${encodeURIComponent(id)}${query({ lock_version: version })}`), { method: "DELETE" }),
  transitionArticle: async (id: string, transition: "publish" | "unpublish" | "archive", version: number, appId?: string | null) => article(await data<AdminContentArticle>(contentPath(appId, `/items/${encodeURIComponent(id)}/${transition}`), body("POST", { lock_version: version }))),
  topics: async (filters: TopicFilters, appId?: string | null) => { const value = await data<ContentList<AdminContentTopic>>(contentPath(appId, `/topics${query(filters)}`)); return { ...value, items: value.items.map(topic) }; },
  createTopic: async (input: TopicInput, appId?: string | null) => topic(await data<AdminContentTopic>(contentPath(appId, "/topics"), body("POST", input))),
  updateTopic: async (id: string, input: TopicInput, appId?: string | null) => topic(await data<AdminContentTopic>(contentPath(appId, `/topics/${id}`), body("PATCH", input))),
  deleteTopic: (id: string, version: number, appId?: string | null) => data<undefined>(contentPath(appId, `/topics/${id}${query({ lock_version: version })}`), { method: "DELETE" }),
  tags: async (filters: TagFilters, appId?: string | null) => { const value = await data<ContentList<AdminContentTag>>(contentPath(appId, `/tags${query(filters)}`)); return { ...value, items: value.items.map(tag) }; },
  createTag: async (name: string, appId?: string | null) => tag(await data<AdminContentTag>(contentPath(appId, "/tags"), body("POST", { name }))),
  updateTag: async (value: ContentTag, appId?: string | null) => tag(await data<AdminContentTag>(contentPath(appId, `/tags/${value.id}`), body("PATCH", { id: value.id, name: value.name, status: value.status, lock_version: value.version }))),
  mergeTag: (id: string, targetId: string, version: number, appId?: string | null) => data<unknown>(contentPath(appId, `/tags/${id}/merge`), body("POST", { target_id: targetId, lock_version: version })),
  deleteTag: (id: string, version: number, appId?: string | null) => data<unknown>(contentPath(appId, `/tags/${id}${query({ lock_version: version })}`), { method: "DELETE" }),
  comments: async (filters: CommentFilters, appId?: string | null) => { const value = await data<ContentList<ContentComment>>(contentPath(appId, `/comments${query(filters)}`)); return { ...value, items: value.items.map(comment) }; },
  moderateComment: async (id: string, status: "approved" | "rejected" | "hidden" | "deleted", reason: string, appId?: string | null) => comment(await data<ContentComment>(contentPath(appId, `/comments/${id}/moderate`), body("POST", { status, reason }))),
  batchModerateComments: async (ids: string[], status: "approved" | "rejected" | "hidden", reason: string, appId?: string | null) => data<{ items: ContentComment[] }>(contentPath(appId, "/comments/batch-moderate"), body("POST", { ids, status, reason })),
  commentReports: async (filters: CommentReportFilters, appId?: string | null) => { const value = await data<ContentList<ContentCommentReport>>(contentPath(appId, `/comment-reports${query(filters)}`)); return { ...value, items: value.items.map(report) }; },
  resolveCommentReport: async (id: string, status: "resolved" | "dismissed", resolution: string, appId?: string | null) => report(await data<ContentCommentReport>(contentPath(appId, `/comment-reports/${id}/resolve`), body("POST", { status, resolution }))),
};
