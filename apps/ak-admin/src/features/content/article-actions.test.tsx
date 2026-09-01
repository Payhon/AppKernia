import { describe, expect, it, vi } from "vitest";
import { createArticleActionItems, type ArticleActionHandlers, type ArticleActionLabels } from "./article-actions";
import type { ContentArticle } from "./model";

const labels: ArticleActionLabels = { preview: "Preview", view: "View", copy: "Copy", edit: "Edit", publish: "Publish", unpublish: "Unpublish", delete: "Delete" };
const handlers: ArticleActionHandlers = { preview: vi.fn(), view: vi.fn(), copy: vi.fn(), edit: vi.fn(), publish: vi.fn(), unpublish: vi.fn(), delete: vi.fn() };
const all = new Set(["app.content.update", "app.content.publish", "app.content.delete"]);
const make = (status: ContentArticle["status"], permissions = all, url: string | null = "https://public.example/page", pending = false, scoped = true) => createArticleActionItems({ status }, permissions, scoped, url, labels, handlers, pending).filter(item => item && item.type !== "divider");
describe("article action menu", () => {
  it.each(["draft", "published", "archived"] as const)("preserves status eligibility and icons for %s", status => {
    const items = make(status);
    expect(items.map(item => item?.key)).toEqual(status === "published" ? ["preview", "view", "copy", "edit", "unpublish", "delete"] : status === "draft" ? ["edit", "publish", "delete"] : ["edit", "delete"]);
    expect(items.every(item => Boolean(item && "icon" in item && item.icon))).toBe(true);
  });
  it("uses the correct permission surface and hides empty menus", () => {
    expect(make("draft", new Set())).toEqual([]);
    expect(make("draft", new Set(["content.article.update"]))).toEqual([]);
    expect(make("draft", new Set(["content.article.update"]), null, false, false).map(item => item?.key)).toEqual(["edit"]);
    expect(make("published", new Set(), null)).toEqual([]);
    expect(make("published", new Set()).map(item => item?.key)).toEqual(["preview", "view", "copy"]);
  });
  it("disables writes while pending but keeps reading available and delete dangerous", () => {
    const items = make("published", all, "https://public.example/page", true);
    for (const item of items) if (item && "disabled" in item && ["edit", "unpublish", "delete"].includes(String(item.key))) expect(item.disabled).toBe(true);
    expect(items.find(item => item?.key === "delete")).toMatchObject({ danger: true });
    expect(items.find(item => item?.key === "preview")).not.toHaveProperty("disabled", true);
  });
});
