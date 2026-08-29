import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { contentApi } from "./api";
import type { ArticleFilters, ArticleInput, CategoryFilters, CategoryInput, CommentFilters, CommentReportFilters, TagFilters, TopicFilters, TopicInput } from "./model";

function useContentQuery<T>(segment: string, filters: object, appId: string | null | undefined, queryFn: () => Promise<T>) { const tenantId = useTenantKey(); return useQuery({ queryKey: ["tenant", tenantId, "content", appId ?? "legacy", segment, filters], queryFn, enabled: appId !== null, placeholderData: (value) => value }); }
export function useContentCategories(filters: CategoryFilters, appId?: string | null) { return useContentQuery("categories", filters, appId, () => contentApi.categories(filters, appId)); }
export function useContentArticles(filters: ArticleFilters, appId?: string | null) { return useContentQuery("items", filters, appId, () => contentApi.articles(filters, appId)); }
export function useContentTopics(filters: TopicFilters, appId?: string | null) { return useContentQuery("topics", filters, appId, () => contentApi.topics(filters, appId)); }
export function useContentTags(filters: TagFilters, appId?: string | null) { return useContentQuery("tags", filters, appId, () => contentApi.tags(filters, appId)); }
export function useContentComments(filters: CommentFilters, appId?: string | null) { return useContentQuery("comments", filters, appId, () => contentApi.comments(filters, appId)); }
export function useContentCommentReports(filters: CommentReportFilters, appId?: string | null) { return useContentQuery("comment-reports", filters, appId, () => contentApi.commentReports(filters, appId)); }

export function useContentMutations(appId?: string | null) {
  const tenantId = useTenantKey(); const client = useQueryClient(); const root = ["tenant", tenantId, "content", appId ?? "legacy"] as const; const invalidate = () => client.invalidateQueries({ queryKey: root });
  return {
    createCategory: useMutation({ mutationFn: (input: CategoryInput) => contentApi.createCategory(input, appId), onSuccess: invalidate }),
    updateCategory: useMutation({ mutationFn: ({ id, input }: { id: string; input: CategoryInput }) => contentApi.updateCategory(id, input, appId), onSuccess: invalidate }),
    deleteCategory: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteCategory(id, version, appId), onSuccess: invalidate }),
    createArticle: useMutation({ mutationFn: (input: ArticleInput) => contentApi.createArticle(input, appId), onSuccess: invalidate }),
    updateArticle: useMutation({ mutationFn: ({ id, input }: { id: string; input: ArticleInput }) => contentApi.updateArticle(id, input, appId), onSuccess: invalidate }),
    deleteArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteArticle(id, version, appId), onSuccess: invalidate }),
    publishArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "publish", version, appId), onSuccess: invalidate }),
    unpublishArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "unpublish", version, appId), onSuccess: invalidate }),
    archiveArticle: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.transitionArticle(id, "archive", version, appId), onSuccess: invalidate }),
    createTopic: useMutation({ mutationFn: (input: TopicInput) => contentApi.createTopic(input, appId), onSuccess: invalidate }),
    updateTopic: useMutation({ mutationFn: ({ id, input }: { id: string; input: TopicInput }) => contentApi.updateTopic(id, input, appId), onSuccess: invalidate }),
    deleteTopic: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteTopic(id, version, appId), onSuccess: invalidate }),
    createTag: useMutation({ mutationFn: (name: string) => contentApi.createTag(name, appId), onSuccess: invalidate }),
    updateTag: useMutation({ mutationFn: (value: Parameters<typeof contentApi.updateTag>[0]) => contentApi.updateTag(value, appId), onSuccess: invalidate }),
    mergeTag: useMutation({ mutationFn: ({ id, targetId, version }: { id: string; targetId: string; version: number }) => contentApi.mergeTag(id, targetId, version, appId), onSuccess: invalidate }),
    deleteTag: useMutation({ mutationFn: ({ id, version }: { id: string; version: number }) => contentApi.deleteTag(id, version, appId), onSuccess: invalidate }),
    moderateComment: useMutation({ mutationFn: ({ id, status, reason }: { id: string; status: "approved" | "rejected" | "hidden" | "deleted"; reason: string }) => contentApi.moderateComment(id, status, reason, appId), onSuccess: invalidate }),
    batchModerateComments: useMutation({ mutationFn: ({ ids, status, reason }: { ids: string[]; status: "approved" | "rejected" | "hidden"; reason: string }) => contentApi.batchModerateComments(ids, status, reason, appId), onSuccess: invalidate }),
    resolveCommentReport: useMutation({ mutationFn: ({ id, status, resolution }: { id: string; status: "resolved" | "dismissed"; resolution: string }) => contentApi.resolveCommentReport(id, status, resolution, appId), onSuccess: invalidate }),
  };
}
