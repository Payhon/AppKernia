import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { contentApi } from "./api";
import type { ArticleFilters, ArticleInput, CategoryFilters, CategoryInput, ContentArticle, ContentCategory, ContentList } from "./model";

export function useContentCategories(filters: CategoryFilters, appId?: string | null) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenantId, "content", appId ?? "legacy", "categories", filters], queryFn: () => contentApi.categories(filters, appId), enabled: appId !== null, placeholderData: (value) => value });
}
export function useContentArticles(filters: ArticleFilters, appId?: string | null) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenantId, "content", appId ?? "legacy", "articles", filters], queryFn: () => contentApi.articles(filters, appId), enabled: appId !== null, placeholderData: (value) => value });
}
export function useContentMutations(appId?: string | null) {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const root = ["tenant", tenantId, "content", appId ?? "legacy"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  const optimistic = <T extends ContentArticle | ContentCategory>(segment: "articles" | "categories", value: T) => {
    const snapshots = client.getQueriesData<ContentList<T>>({ queryKey: [...root, segment] });
    client.setQueriesData<ContentList<T>>({ queryKey: [...root, segment] }, (previous) => previous ? { ...previous, items: previous.items.map((item) => item.id === value.id ? value : item) } : previous);
    return snapshots;
  };
  const restore = <T extends ContentArticle | ContentCategory>(snapshots: readonly [readonly unknown[], ContentList<T> | undefined][]) => { snapshots.forEach(([key, value]) => client.setQueryData(key, value)); };
  return {
    createCategory: useMutation({ mutationFn: (input: CategoryInput) => contentApi.createCategory(input, appId), onSuccess: invalidate }),
    updateCategory: useMutation({ mutationFn: ({ id, input }: { id: string; input: CategoryInput }) => contentApi.updateCategory(id, input, appId), onMutate: async ({ id, input }) => { await client.cancelQueries({ queryKey: [...root, "categories"] }); return optimistic("categories", { ...input, id, updated_at: "", version: input.version ?? 0 }); }, onError: (_error, _value, context) => { if (context) restore(context); }, onSettled: invalidate }),
    deleteCategory: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteCategory(id, version, appId), onSuccess: invalidate }),
    createArticle: useMutation({ mutationFn: (input: ArticleInput) => contentApi.createArticle(input, appId), onSuccess: invalidate }),
    updateArticle: useMutation({ mutationFn: ({ id, input }: { id: string; input: ArticleInput; status: ContentArticle["status"] }) => contentApi.updateArticle(id, input, appId), onMutate: async ({ id, input, status }) => { await client.cancelQueries({ queryKey: [...root, "articles"] }); return optimistic("articles", { ...input, id, status, cover_url: null, published_at: null, updated_at: "", version: input.version ?? 0 }); }, onError: (_error, _value, context) => { if (context) restore(context); }, onSettled: invalidate }),
    deleteArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteArticle(id, version, appId), onSuccess: invalidate }),
    publishArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "publish", version, appId), onSuccess: invalidate }),
    unpublishArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "unpublish", version, appId), onSuccess: invalidate }),
    archiveArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "archive", version, appId), onSuccess: invalidate }),
  };
}
