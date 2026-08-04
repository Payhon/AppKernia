import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AdminWebhookRequest, AdminWebhookTestRequest } from "../../generated/api/types.gen";
import { authSession } from "../auth/store";
import type { AdminWebhookFilters } from "../auth/session";
import { useTenantKey } from "../tenants/hooks";

export function useWebhooks(filters: AdminWebhookFilters) {
  const tenant = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenant, "webhooks", filters], queryFn: () => authSession.adminWebhooks(filters), placeholderData: (value) => value });
}
export function useWebhookDeliveries(id: string, page: number, pageSize: number) {
  const tenant = useTenantKey();
  return useQuery({ queryKey: ["tenant", tenant, "webhooks", id, "deliveries", page, pageSize], queryFn: () => authSession.adminWebhookDeliveries(id, page, pageSize), enabled: Boolean(id), placeholderData: (value) => value });
}
export function useWebhookMutations() {
  const tenant = useTenantKey();
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["tenant", tenant, "webhooks"] });
  return {
    create: useMutation({ mutationFn: (input: AdminWebhookRequest) => authSession.createAdminWebhook(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminWebhookRequest }) => authSession.updateAdminWebhook(id, input), onSuccess: invalidate }),
    test: useMutation({ mutationFn: ({ id, key, input }: { id: string; key: string; input: AdminWebhookTestRequest }) => authSession.testAdminWebhook(id, key, input), onSuccess: invalidate }),
  };
}
