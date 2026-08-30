import type {
  AdminNotificationFailurePageResponse,
  AdminNotificationOperationsSummaryResponse,
  AdminNotificationRunPageResponse,
  AdminNotificationRunResponse,
  AdminNotificationTaskPageResponse,
  AdminNotificationTaskResponse,
  AdminNotificationTaskRetryRequest,
  AdminNotificationTaskRetryResponse,
  AdminNotificationTrendListResponse,
} from "../../generated/api/types.gen";
import { toApiError } from "../../shared/api/error";
import { authSession } from "../auth/store";
import { operationsTimeRange, type OperationsFilters } from "./model";

async function json<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return (await response.json()) as T;
}

function query(filters: OperationsFilters, includePage = false) {
  const params = new URLSearchParams(operationsTimeRange(filters.range));
  for (const key of ["environment", "category", "channel", "provider", "task_kind", "status", "q"] as const) {
    const value = filters[key];
    if (value) params.set(key, value);
  }
  if (includePage) {
    params.set("page", String(filters.page));
    params.set("page_size", String(filters.page_size));
  }
  return params.toString();
}

const root = (appId: string) => `/apps/${encodeURIComponent(appId)}` as const;

export async function getOperationsSummary(appId: string, filters: OperationsFilters) {
  return (await json<AdminNotificationOperationsSummaryResponse>(`${root(appId)}/notification-operations/summary?${query(filters)}`)).data;
}

export async function getOperationsTrends(appId: string, filters: OperationsFilters) {
  return (await json<AdminNotificationTrendListResponse>(`${root(appId)}/notification-operations/trends?${query(filters)}`)).data.items;
}

export async function listNotificationRuns(appId: string, filters: OperationsFilters) {
  return (await json<AdminNotificationRunPageResponse>(`${root(appId)}/notification-runs?${query(filters, true)}`)).data;
}

export async function getNotificationRun(appId: string, runId: string) {
  return (await json<AdminNotificationRunResponse>(`${root(appId)}/notification-runs/${encodeURIComponent(runId)}`)).data;
}

export async function listNotificationTasks(appId: string, filters: OperationsFilters) {
  return (await json<AdminNotificationTaskPageResponse>(`${root(appId)}/notification-tasks?${query(filters, true)}`)).data;
}

export async function getNotificationTask(appId: string, taskId: string) {
  return (await json<AdminNotificationTaskResponse>(`${root(appId)}/notification-tasks/${encodeURIComponent(taskId)}`)).data;
}

export async function listNotificationFailures(appId: string, filters: OperationsFilters) {
  return (await json<AdminNotificationFailurePageResponse>(`${root(appId)}/notification-failures?${query(filters, true)}`)).data;
}

export async function retryNotificationTasks(appId: string, input: AdminNotificationTaskRetryRequest) {
  return (await json<AdminNotificationTaskRetryResponse>(`${root(appId)}/notification-retries`, {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input),
  })).data.items;
}
