import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { mobileReleasesApi } from "./api";
import type { MobileReleaseInput } from "./model";

export function useMobileReleases() {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "mobile-releases"],
    queryFn: mobileReleasesApi.list,
  });
}

export function useMobileReleaseMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const key = ["tenant", tenantId, "mobile-releases"] as const;
  const invalidate = () => client.invalidateQueries({ queryKey: key });
  return {
    create: useMutation({ mutationFn: (input: MobileReleaseInput) => mobileReleasesApi.create(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: MobileReleaseInput }) => mobileReleasesApi.update(id, input), onSuccess: invalidate }),
  };
}
