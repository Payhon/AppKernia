import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  AdminNotificationMessageRequest,
  AdminNotificationTemplateRequest,
  AdminNotificationTemplateTestRequest,
  AdminSmsTemplateBindingRequest,
} from "../../generated/api/types.gen";
import {
  type AdminNotificationDeliveryFilters,
  type AdminNotificationMessageFilters,
  type AdminNotificationTemplateFilters,
} from "../auth/session";
import { authSession } from "../auth/store";
import { useTenantKey } from "../tenants/hooks";

export type NotificationKind = "notices" | "messages";

export function useNotificationMessages(
  kind: NotificationKind,
  filters: AdminNotificationMessageFilters,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", kind, filters],
    queryFn: () => authSession.adminNotificationMessages(kind, filters),
    placeholderData: (value) => value,
  });
}

export function useNotificationMessage(
  kind: NotificationKind,
  id: string | null,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", kind, id],
    queryFn: () => {
      if (!id) throw new Error("notification id is required");
      return authSession.adminNotificationMessage(kind, id);
    },
    enabled: Boolean(id),
  });
}

export function useNotificationRecipientStats(
  kind: NotificationKind,
  id: string | null,
  enabled = true,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", kind, id, "recipients"],
    queryFn: () => {
      if (!id) throw new Error("notification id is required");
      return authSession.adminNotificationRecipientStats(kind, id);
    },
    enabled: Boolean(id) && enabled,
  });
}

export function useNotificationMessageMutations(kind: NotificationKind) {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const invalidate = () =>
    client.invalidateQueries({
      queryKey: ["tenant", tenantId, "notifications", kind],
    });
  return {
    create: useMutation({
      mutationFn: (input: AdminNotificationMessageRequest) =>
        authSession.createAdminNotificationMessage(kind, input),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: (value: {
        id: string;
        input: AdminNotificationMessageRequest;
      }) =>
        authSession.updateAdminNotificationMessage(
          kind,
          value.id,
          value.input,
        ),
      onSuccess: invalidate,
    }),
    publish: useMutation({
      mutationFn: (id: string) =>
        authSession.publishAdminNotificationMessage(kind, id),
      onSuccess: invalidate,
    }),
    cancel: useMutation({
      mutationFn: (id: string) =>
        authSession.cancelAdminNotificationMessage(kind, id),
      onSuccess: invalidate,
    }),
  };
}

export function useNotificationTemplates(
  filters: AdminNotificationTemplateFilters,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", "templates", filters],
    queryFn: () => authSession.adminNotificationTemplates(filters),
    placeholderData: (value) => value,
  });
}

export function useNotificationTemplateMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const invalidate = () =>
    client.invalidateQueries({
      queryKey: ["tenant", tenantId, "notifications", "templates"],
    });
  return {
    create: useMutation({
      mutationFn: (input: AdminNotificationTemplateRequest) =>
        authSession.createAdminNotificationTemplate(input),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: (value: {
        id: string;
        input: AdminNotificationTemplateRequest;
      }) => authSession.updateAdminNotificationTemplate(value.id, value.input),
      onSuccess: invalidate,
    }),
    upsertBinding: useMutation({
      mutationFn: (value: {
        id: string;
        provider: "aliyun" | "tencent";
        input: AdminSmsTemplateBindingRequest;
      }) => authSession.upsertAdminSmsTemplateBinding(value.id, value.provider, value.input),
      onSuccess: invalidate,
    }),
    deleteBinding: useMutation({
      mutationFn: (value: { id: string; provider: "aliyun" | "tencent" }) =>
        authSession.deleteAdminSmsTemplateBinding(value.id, value.provider),
      onSuccess: invalidate,
    }),
    test: useMutation({
      mutationFn: (value: { id: string; input: AdminNotificationTemplateTestRequest }) =>
        authSession.testAdminNotificationTemplate(value.id, value.input),
    }),
  };
}

export function useSmsTemplateBindings(id: string | null) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", "templates", id, "sms-bindings"],
    queryFn: () => {
      if (!id) throw new Error("template id is required");
      return authSession.adminSmsTemplateBindings(id);
    },
    enabled: Boolean(id),
  });
}

export function useNotificationDeliveries(
  filters: AdminNotificationDeliveryFilters,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", "deliveries", filters],
    queryFn: () => authSession.adminNotificationDeliveries(filters),
    placeholderData: (value) => value,
  });
}

export function useNotificationDelivery(id: string | null) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "notifications", "deliveries", id],
    queryFn: () => {
      if (!id) throw new Error("delivery id is required");
      return authSession.adminNotificationDelivery(id);
    },
    enabled: Boolean(id),
  });
}

export function useNotificationDeliveryMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  return {
    retry: useMutation({
      mutationFn: (value: { id: string; acknowledge_duplicate_risk?: boolean }) =>
        authSession.retryAdminNotificationDelivery(value.id, {
          acknowledge_duplicate_risk: value.acknowledge_duplicate_risk ?? false,
        }),
      onSuccess: () =>
        client.invalidateQueries({
          queryKey: ["tenant", tenantId, "notifications", "deliveries"],
        }),
    }),
  };
}
