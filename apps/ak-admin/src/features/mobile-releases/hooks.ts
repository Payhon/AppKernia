import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { mobileReleasesApi } from "./api";
import type { MobileReleaseFilters, MobileReleaseInput } from "./model";

const rootKey = (tenantId: string, appId: string | null) => ["tenant", tenantId, "applications", appId ?? "none", "mobile-releases"] as const;
export function useMobileReleases(appId: string | null, filters: MobileReleaseFilters) {
  const tenantId = useTenantKey();
  return useQuery({ queryKey: [...rootKey(tenantId, appId), "list", filters], queryFn: () => appId ? mobileReleasesApi.list(appId, filters) : Promise.reject(new Error("APP_SCOPE_REQUIRED")), enabled: appId !== null, placeholderData: (value) => value });
}
export function useMobileReleaseMutations(appId: string | null) {
  const tenantId = useTenantKey(); const client = useQueryClient();
  const invalidate = () => client.invalidateQueries({ queryKey: rootKey(tenantId, appId) });
  const requireApp = () => { if (!appId) throw new Error("APP_SCOPE_REQUIRED"); return appId; };
  return {
    create: useMutation({ mutationFn: (input: MobileReleaseInput) => mobileReleasesApi.create(requireApp(), input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: MobileReleaseInput }) => mobileReleasesApi.update(requireApp(), id, input), onSuccess: invalidate }),
    publish: useMutation({ mutationFn: ({ id, lockVersion }: { id: string; lockVersion: number }) => mobileReleasesApi.action(requireApp(), id, "publish", lockVersion), onSuccess: invalidate }),
    unpublish: useMutation({ mutationFn: ({ id, lockVersion }: { id: string; lockVersion: number }) => mobileReleasesApi.action(requireApp(), id, "unpublish", lockVersion), onSuccess: invalidate }),
    delete: useMutation({ mutationFn: (id: string) => mobileReleasesApi.delete(requireApp(), id), onSuccess: invalidate }),
    batchDelete: useMutation({ mutationFn: (ids: string[]) => mobileReleasesApi.batchDelete(requireApp(), ids), onSuccess: invalidate }),
  };
}
