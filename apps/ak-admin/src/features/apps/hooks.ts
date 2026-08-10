import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { appAdminApi } from "./api";
import type { AppListFilters, AppMemberCreateInput, AppMemberUpdateInput, ApplicationInput, AppPageInput, MemberListFilters, PageListFilters } from "./model";

const rootKey = (tenantId: string) => ["tenant", tenantId, "applications"] as const;

export function useManagedApplications(filters: AppListFilters) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: [...rootKey(tenantId), "list", filters], queryFn: () => appAdminApi.list(filters), placeholderData: (value) => value });
}
export function useAppMembers(appId: string | null, filters: MemberListFilters) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: [...rootKey(tenantId), appId ?? "none", "members", filters], queryFn: () => appId ? appAdminApi.members(appId, filters) : Promise.reject(new Error("APP_SCOPE_REQUIRED")), enabled: appId !== null, placeholderData: (value) => value });
}
export function useAppPages(appId: string | null, filters: PageListFilters) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: [...rootKey(tenantId), appId ?? "none", "pages", filters], queryFn: () => appId ? appAdminApi.pages(appId, filters) : Promise.reject(new Error("APP_SCOPE_REQUIRED")), enabled: appId !== null, placeholderData: (value) => value });
}
export function useApplicationMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const invalidate = () => client.invalidateQueries({ queryKey: rootKey(tenantId) });
  return {
    create: useMutation({ mutationFn: (input: ApplicationInput) => appAdminApi.create(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: ApplicationInput }) => appAdminApi.update(id, input), onSuccess: invalidate }),
    delete: useMutation({ mutationFn: (id: string) => appAdminApi.delete(id), onSuccess: invalidate }),
    batchDelete: useMutation({ mutationFn: (ids: string[]) => appAdminApi.batchDelete(ids), onSuccess: invalidate }),
    status: useMutation({ mutationFn: ({ id, action, lockVersion }: { id: string; action: "enable" | "disable"; lockVersion: number }) => appAdminApi.setStatus(id, action, lockVersion), onSuccess: invalidate }),
    createMember: useMutation({ mutationFn: ({ appId, input }: { appId: string; input: AppMemberCreateInput }) => appAdminApi.createMember(appId, input), onSuccess: invalidate }),
    updateMember: useMutation({ mutationFn: ({ appId, memberId, input }: { appId: string; memberId: string; input: AppMemberUpdateInput }) => appAdminApi.updateMember(appId, memberId, input), onSuccess: invalidate }),
    memberAction: useMutation({ mutationFn: ({ appId, memberId, action, lockVersion }: { appId: string; memberId: string; action: "enable" | "disable" | "unlock" | "revoke-sessions"; lockVersion: number }) => appAdminApi.memberAction(appId, memberId, action, lockVersion), onSuccess: invalidate }),
    resetMemberPassword: useMutation({ mutationFn: ({ appId, memberId, newPassword, lockVersion }: { appId: string; memberId: string; newPassword: string; lockVersion: number }) => appAdminApi.resetMemberPassword(appId, memberId, newPassword, lockVersion), onSuccess: invalidate }),
    createPage: useMutation({ mutationFn: ({ appId, input }: { appId: string; input: AppPageInput }) => appAdminApi.createPage(appId, input), onSuccess: invalidate }),
    updatePage: useMutation({ mutationFn: ({ appId, pageId, input }: { appId: string; pageId: string; input: AppPageInput }) => appAdminApi.updatePage(appId, pageId, input), onSuccess: invalidate }),
    publishPage: useMutation({ mutationFn: ({ appId, pageId, lockVersion }: { appId: string; pageId: string; lockVersion: number }) => appAdminApi.publishPage(appId, pageId, lockVersion), onSuccess: invalidate }),
    deletePage: useMutation({ mutationFn: ({ appId, pageId, lockVersion }: { appId: string; pageId: string; lockVersion: number }) => appAdminApi.deletePage(appId, pageId, lockVersion), onSuccess: invalidate }),
  };
}
