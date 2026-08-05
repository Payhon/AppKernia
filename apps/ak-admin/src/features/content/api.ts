import { authSession } from "../auth/store";
import { toApiError } from "../../shared/api/error";
import type { AdminContentArticle, AdminContentArticleRequest, AdminContentCategory, AdminContentCategoryRequest } from "../../generated/api/types.gen";
import type { ArticleFilters, ArticleInput, CategoryFilters, CategoryInput, ContentArticle, ContentCategory, ContentList } from "./model";

async function data<T>(path: `/${string}`, init?: RequestInit): Promise<T> { const response = await authSession.adminRequest(path, init); if (!response.ok) throw await toApiError(response); return ((await response.json()) as { data: T }).data; }
function query(filters: object) { const params = new URLSearchParams(); for (const [key, value] of Object.entries(filters)) if ((typeof value === "string" || typeof value === "number") && value !== "") params.set(key, String(value)); return params.size ? `?${params}` : ""; }
const json = (body: unknown): RequestInit => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const patch = (body: unknown): RequestInit => ({ method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const categoryRequest = (input: CategoryInput): AdminContentCategoryRequest => ({ slug: input.slug, sort_order: input.sort_order, status: input.status, translations: input.translations, ...(input.version ? { lock_version: input.version } : {}) });
const articleRequest = (input: ArticleInput): AdminContentArticleRequest => ({
  slug: input.slug,
  category_id: input.category_id || null,
  featured: input.featured,
  sort_order: input.sort_order,
  cover_file_id: input.cover_file_id,
  reading_minutes: input.reading_minutes,
  translations: Object.fromEntries(Object.entries(input.translations).map(([locale, translation]) => [locale, {
    ...translation,
    body: translation.body_format === "blocks" ? JSON.parse(translation.body) as unknown : translation.body,
  }])) as AdminContentArticleRequest["translations"],
  ...(input.version ? { lock_version: input.version } : {}),
});
const category = (value: AdminContentCategory): ContentCategory => ({ ...value, version: value.lock_version });
const article = (value: AdminContentArticle): ContentArticle => ({
  ...value,
  category_id: value.category_id ?? "",
  cover_file_id: value.cover_file_id ?? null,
  cover_url: value.cover_url ?? null,
  published_at: value.published_at ?? null,
  translations: Object.fromEntries(Object.entries(value.translations).map(([locale, translation]) => [locale, {
    ...translation,
    body: translation.body_format === "blocks" ? JSON.stringify(translation.body) : String(translation.body),
  }])) as ContentArticle["translations"],
  version: value.lock_version,
});

export const contentApi = {
  categories: async (filters: CategoryFilters) => { const value = await data<ContentList<AdminContentCategory>>(`/content/categories${query(filters)}`); return { ...value, items: value.items.map(category) }; },
  createCategory: async (input: CategoryInput) => category(await data<AdminContentCategory>("/content/categories", json(categoryRequest(input)))),
  updateCategory: async (id: string, input: CategoryInput) => category(await data<AdminContentCategory>(`/content/categories/${encodeURIComponent(id)}`, patch(categoryRequest(input)))),
  deleteCategory: (id: string, version: number) => data<undefined>(`/content/categories/${encodeURIComponent(id)}${query({ lock_version: version })}`, { method: "DELETE" }),
  articles: async (filters: ArticleFilters) => { const value = await data<ContentList<AdminContentArticle>>(`/content/articles${query(filters)}`); return { ...value, items: value.items.map(article) }; },
  createArticle: async (input: ArticleInput) => article(await data<AdminContentArticle>("/content/articles", json(articleRequest(input)))),
  updateArticle: async (id: string, input: ArticleInput) => article(await data<AdminContentArticle>(`/content/articles/${encodeURIComponent(id)}`, patch(articleRequest(input)))),
  deleteArticle: (id: string, version: number) => data<undefined>(`/content/articles/${encodeURIComponent(id)}${query({ lock_version: version })}`, { method: "DELETE" }),
  transitionArticle: async (id: string, transition: "publish" | "unpublish" | "archive", version: number) => article(await data<AdminContentArticle>(`/content/articles/${encodeURIComponent(id)}/${transition}`, json({ lock_version: version }))),
};
