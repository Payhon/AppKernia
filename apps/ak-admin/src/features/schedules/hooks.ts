import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AdminJobScheduleRequest } from "../../generated/api/types.gen";
import type {
  AdminJobRunFilters,
  AdminJobScheduleFilters,
} from "../auth/session";
import { authSession } from "../auth/store";
import { useTenantKey } from "../tenants/hooks";

export function useJobHandlers() {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "jobs", "handlers"],
    queryFn: () => authSession.adminJobHandlers(),
    staleTime: Number.POSITIVE_INFINITY,
  });
}

export function useJobSchedules(filters: AdminJobScheduleFilters) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "jobs", "schedules", filters],
    queryFn: () => authSession.adminJobSchedules(filters),
    placeholderData: (value) => value,
  });
}

export function useJobScheduleRuns(
  id: string | null,
  filters: AdminJobRunFilters,
) {
  const tenantId = useTenantKey();
  return useQuery({
    queryKey: ["tenant", tenantId, "jobs", "schedules", id, "runs", filters],
    queryFn: () => {
      if (!id) throw new Error("schedule id is required");
      return authSession.adminJobScheduleRuns(id, filters);
    },
    enabled: Boolean(id),
    refetchInterval: (query) =>
      query.state.data?.items.some((item) =>
        ["queued", "running"].includes(item.status),
      )
        ? 1000
        : false,
  });
}

export function useJobScheduleMutations() {
  const tenantId = useTenantKey();
  const client = useQueryClient();
  const invalidateSchedules = () =>
    client.invalidateQueries({
      queryKey: ["tenant", tenantId, "jobs", "schedules"],
    });
  return {
    preview: useMutation({
      mutationFn: (input: AdminJobScheduleRequest) =>
        authSession.previewAdminJobSchedule(input),
    }),
    create: useMutation({
      mutationFn: (input: AdminJobScheduleRequest) =>
        authSession.createAdminJobSchedule(input),
      onSuccess: invalidateSchedules,
    }),
    update: useMutation({
      mutationFn: (value: { id: string; input: AdminJobScheduleRequest }) =>
        authSession.updateAdminJobSchedule(value.id, value.input),
      onSuccess: invalidateSchedules,
    }),
    pause: useMutation({
      mutationFn: (value: { id: string; paused: boolean }) =>
        authSession.setAdminJobSchedulePaused(value.id, value.paused),
      onSuccess: invalidateSchedules,
    }),
    execute: useMutation({
      mutationFn: (value: { id: string; idempotencyKey: string }) =>
        authSession.executeAdminJobSchedule(value.id, value.idempotencyKey),
      onSuccess: invalidateSchedules,
    }),
  };
}
