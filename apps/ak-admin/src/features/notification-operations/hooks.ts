import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTenantKey } from "../tenants/hooks";
import { getNotificationRun, getNotificationTask, getOperationsSummary, getOperationsTrends, listNotificationFailures, listNotificationRuns, listNotificationTasks, retryNotificationTasks } from "./api";
import type { OperationsFilters } from "./model";

export function useNotificationOperations(appId: string | null, filters: OperationsFilters) {
  const tenant = useTenantKey();
  const root = ["tenant", tenant, "apps", appId, "notification-operations"] as const;
  const requireApp = () => { if (!appId) throw new Error("App selection is required"); return appId; };
  const summary = useQuery({ queryKey: [...root, "summary", filters], queryFn: () => getOperationsSummary(requireApp(), filters), enabled: Boolean(appId), refetchInterval: (query) => document.visibilityState === "visible" && query.state.data?.has_unfinished_work ? 15_000 : false });
  const pollWhileActive = () => document.visibilityState === "visible" && summary.data?.has_unfinished_work ? 15_000 : false;
  const shared = { enabled: Boolean(appId), placeholderData: keepPreviousData, refetchInterval: pollWhileActive };
  return {
    root,
    summary,
    trends: useQuery({ queryKey: [...root, "trends", filters], queryFn: () => getOperationsTrends(requireApp(), filters), ...shared }),
    runs: useQuery({ queryKey: [...root, "runs", filters], queryFn: () => listNotificationRuns(requireApp(), filters), ...shared, enabled: Boolean(appId) && filters.tab === "runs" }),
    tasks: useQuery({ queryKey: [...root, "tasks", filters], queryFn: () => listNotificationTasks(requireApp(), filters), ...shared, enabled: Boolean(appId) && filters.tab === "tasks" }),
    failures: useQuery({ queryKey: [...root, "failures", filters], queryFn: () => listNotificationFailures(requireApp(), filters), ...shared, enabled: Boolean(appId) && filters.tab === "failures" }),
  };
}

export function useNotificationRun(appId: string | null, runId: string | null) {
  return useQuery({ queryKey: ["notification-run", appId, runId], queryFn: () => {
    if (!appId || !runId) throw new Error("App and run selections are required");
    return getNotificationRun(appId, runId);
  }, enabled: Boolean(appId && runId) });
}

export function useNotificationTask(appId: string | null, taskId: string | null) {
  return useQuery({ queryKey: ["notification-task", appId, taskId], queryFn: () => {
    if (!appId || !taskId) throw new Error("App and task selections are required");
    return getNotificationTask(appId, taskId);
  }, enabled: Boolean(appId && taskId) });
}

export function useNotificationRetry(appId: string | null, queryRoot: readonly unknown[]) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ taskIds, acknowledge }: { taskIds: string[]; acknowledge: boolean }) => {
      if (!appId) throw new Error("App selection is required");
      return retryNotificationTasks(appId, { items: taskIds.map((task_id) => ({ task_id })), acknowledge_duplicate_risk: acknowledge });
    },
    onSuccess: () => client.invalidateQueries({ queryKey: queryRoot }),
  });
}
