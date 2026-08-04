import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AdminBlockRuleCreateRequestWritable, AdminBlockRuleUpdateRequest } from "../../generated/api/types.gen";
import { authSession } from "../auth/store";
import type { AdminBlockRuleFilters } from "../auth/session";
import { useTenantKey } from "../tenants/hooks";

export function useBlockRules(filters: AdminBlockRuleFilters) {
  const tenant = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenant, "block-rules", filters], queryFn: () => authSession.adminBlockRules(filters), placeholderData: (value) => value });
}
export function useBlockRuleMutations() {
  const tenant = useTenantKey();
  const client = useQueryClient();
  const invalidate = () => client.invalidateQueries({ queryKey: ["tenant", tenant, "block-rules"] });
  return {
    create: useMutation({ mutationFn: (input: AdminBlockRuleCreateRequestWritable) => authSession.createAdminBlockRule(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminBlockRuleUpdateRequest }) => authSession.updateAdminBlockRule(id, input), onSuccess: invalidate }),
    revoke: useMutation({ mutationFn: (id: string) => authSession.revokeAdminBlockRule(id), onSuccess: invalidate }),
  };
}
