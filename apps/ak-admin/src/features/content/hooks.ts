import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { contentApi } from "./api";
import type { ArticleFilters, ArticleInput, CategoryFilters, CategoryInput, ContentArticle, ContentCategory, ContentList } from "./model";

export function useContentCategories(filters: CategoryFilters) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenantId, "content", "categories", filters], queryFn: () => contentApi.categories(filters), placeholderData: (value) => value });
}
export function useContentArticles(filters: ArticleFilters) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenantId, "content", "articles", filters], queryFn: () => contentApi.articles(filters), placeholderData: (value) => value });
}
export function useContentMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const root = ["tenant", tenantId, "content"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: root });
  const optimistic = <T extends ContentArticle | ContentCategory>(segment: "articles" | "categories", value: T) => {
    const snapshots = client.getQueriesData<ContentList<T>>({ queryKey: [...root, segment] });
    client.setQueriesData<ContentList<T>>({ queryKey: [...root, segment] }, (previous) => previous ? { ...previous, items: previous.items.map((item) => item.id === value.id ? value : item) } : previous);
    return snapshots;
  };
  const restore = <T extends ContentArticle | ContentCategory>(snapshots: readonly [readonly unknown[], ContentList<T> | undefined][]) => { snapshots.forEach(([key, value]) => client.setQueryData(key, value)); };
  return {
    createCategory: useMutation({ mutationFn: (input: CategoryInput) => contentApi.createCategory(input), onSuccess: invalidate }),
    updateCategory: useMutation({ mutationFn: ({ id, input }: { id: string; input: CategoryInput }) => contentApi.updateCategory(id, input), onMutate: async ({ id, input }) => { await client.cancelQueries({ queryKey: [...root, "categories"] }); return optimistic("categories", { ...input, id, updated_at: "", version: input.version ?? 0 }); }, onError: (_error, _value, context) => { if (context) restore(context); }, onSettled: invalidate }),
    deleteCategory: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteCategory(id, version), onSuccess: invalidate }),
    createArticle: useMutation({ mutationFn: (input: ArticleInput) => contentApi.createArticle(input), onSuccess: invalidate }),
    updateArticle: useMutation({ mutationFn: ({ id, input }: { id: string; input: ArticleInput; status: ContentArticle["status"] }) => contentApi.updateArticle(id, input), onMutate: async ({ id, input, status }) => { await client.cancelQueries({ queryKey: [...root, "articles"] }); return optimistic("articles", { ...input, id, status, cover_url: null, published_at: null, updated_at: "", version: input.version ?? 0 }); }, onError: (_error, _value, context) => { if (context) restore(context); }, onSettled: invalidate }),
    deleteArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteArticle(id, version), onSuccess: invalidate }),
    publishArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "publish", version), onSuccess: invalidate }),
    unpublishArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "unpublish", version), onSuccess: invalidate }),
    archiveArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "archive", version), onSuccess: invalidate }),
  };
}
