import { z } from "zod";

export const contentStatusSchema = z.enum(["draft", "published", "archived"]);
export type ContentStatus = z.infer<typeof contentStatusSchema>;

export interface ContentCategory {
  id: string;
  slug: string;
  sort_order: number;
  status: "active" | "disabled";
  version: number;
  updated_at: string;
  translations: Record<"zh-CN" | "en-US", { name: string; description: string }>;
}

export interface ContentArticle {
  id: string;
  slug: string;
  category_id: string;
  cover_file_id: string | null;
  cover_url: string | null;
  reading_minutes: number;
  featured: boolean;
  sort_order: number;
  status: ContentStatus;
  published_at: string | null;
  version: number;
  updated_at: string;
  translations: Record<"zh-CN" | "en-US", { title: string; summary: string; body_format: "markdown" | "blocks"; body: string }>;
}

export interface ContentList<T> { items: T[]; total: number; }

export const categoryInputSchema = z.object({
  slug: z.string().trim().min(1).max(160).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
  translations: z.object({
    "zh-CN": z.object({ name: z.string().trim().min(1).max(160), description: z.string().trim().max(500) }),
    "en-US": z.object({ name: z.string().trim().min(1).max(160), description: z.string().trim().max(500) }),
  }),
  sort_order: z.number().int().min(0).max(99999),
  status: z.enum(["active", "disabled"]),
  version: z.number().int().positive().optional(),
});
export type CategoryInput = z.infer<typeof categoryInputSchema>;

export const articleInputSchema = z.object({
  slug: z.string().trim().min(1).max(200).regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
  category_id: z.uuid(),
  cover_file_id: z.uuid().nullable(),
  reading_minutes: z.number().int().min(1).max(120),
  featured: z.boolean(),
  sort_order: z.number().int().min(0).max(99999),
  translations: z.object({
    "zh-CN": z.object({ title: z.string().trim().min(1).max(300), summary: z.string().trim().max(1000), body_format: z.enum(["markdown", "blocks"]), body: z.string().trim().min(1).max(100000) }),
    "en-US": z.object({ title: z.string().trim().min(1).max(300), summary: z.string().trim().max(1000), body_format: z.enum(["markdown", "blocks"]), body: z.string().trim().min(1).max(100000) }),
  }),
  version: z.number().int().positive().optional(),
}).superRefine((value, context) => {
  for (const locale of ["zh-CN", "en-US"] as const) {
    const translation = value.translations[locale];
    if (translation.body_format !== "blocks") continue;
    try {
      if (!Array.isArray(JSON.parse(translation.body))) throw new Error("blocks body must be an array");
    } catch {
      context.addIssue({
        code: "custom",
        path: ["translations", locale, "body"],
        message: "blocks body must be a JSON array",
      });
    }
  }
});
export type ArticleInput = z.infer<typeof articleInputSchema>;

export interface ArticleFilters { q: string; category_id: string; status: string; page: number; page_size: number; sort: "updated_desc" | "published_desc" | "sort_order" | "slug"; }
export interface CategoryFilters { q: string; status: string; page: number; page_size: number; sort: "sort_order" | "updated_desc" | "slug"; }
