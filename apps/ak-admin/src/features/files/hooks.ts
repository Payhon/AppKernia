import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authSession } from "../auth/store";
import type { AdminFileFilters } from "../auth/session";
import { useTenantKey } from "../tenants/hooks";

export function useAdminFiles(filters: AdminFileFilters) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "files", filters],
    queryFn: () => authSession.adminFiles(filters),
    placeholderData: (value) => value,
  });
}

export function useAdminFile(id: string | null) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "files", id],
    queryFn: () => {
      if (id === null) throw new Error("file id is required");
      return authSession.adminFile(id);
    },
    enabled: Boolean(id),
  });
}

export function useAdminFileUsages(id: string | null) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "files", id, "usages"],
    queryFn: () => {
      if (id === null) throw new Error("file id is required");
      return authSession.adminFileUsages(id);
    },
    enabled: Boolean(id),
  });
}

export function useAdminFileMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const invalidate = () =>
    client.invalidateQueries({ queryKey: ["tenant", tenantId, "files"] });
  return {
    remove: useMutation({
      mutationFn: (id: string) => authSession.deleteAdminFile(id),
      onSuccess: invalidate,
    }),
  };
}
