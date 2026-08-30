import type { PushEnvironment, PushWritableProvider } from "../../generated/api/types.gen";

export type OperationsTab = "overview" | "runs" | "tasks" | "failures";
export type OperationsRange = "7d" | "30d" | "90d";

export interface OperationsFilters {
  tab: OperationsTab;
  range: OperationsRange;
  environment: "" | PushEnvironment;
  category: "" | "service_security" | "news_operations";
  channel: "" | "in_app" | "email" | "sms" | "push" | "webhook";
  provider: "" | PushWritableProvider;
  task_kind: string;
  status: string;
  q: string;
  page: number;
  page_size: number;
}

const tabs: OperationsTab[] = ["overview", "runs", "tasks", "failures"];
const ranges: OperationsRange[] = ["7d", "30d", "90d"];
const environments = ["", "development", "test", "staging", "production"] as const;
const categories = ["", "service_security", "news_operations"] as const;
const channels = ["", "in_app", "email", "sms", "push", "webhook"] as const;

export function readOperationsFilters(search: string): OperationsFilters {
  const params = new URLSearchParams(search);
  const tab = params.get("tab") as OperationsTab;
  const range = params.get("range") as OperationsRange;
  const environment = params.get("environment") as OperationsFilters["environment"];
  const category = params.get("category") as OperationsFilters["category"];
  const channel = params.get("channel") as OperationsFilters["channel"];
  const rawPage = Number(params.get("page") ?? 1);
  const rawSize = Number(params.get("page_size") ?? 20);
  return {
    tab: tabs.includes(tab) ? tab : "overview",
    range: ranges.includes(range) ? range : "30d",
    environment: environments.includes(environment) ? environment : "",
    category: categories.includes(category) ? category : "",
    channel: channels.includes(channel) ? channel : "",
    provider: (params.get("provider") ?? "") as OperationsFilters["provider"],
    task_kind: params.get("task_kind") ?? "",
    status: params.get("status") ?? "",
    q: params.get("q") ?? "",
    page: Number.isInteger(rawPage) && rawPage > 0 ? rawPage : 1,
    page_size: [10, 20, 50, 100].includes(rawSize) ? rawSize : 20,
  };
}

export function persistOperationsFilters(filters: OperationsFilters, appId: string | null) {
  const params = new URLSearchParams();
  if (appId) params.set("app_id", appId);
  if (filters.tab !== "overview") params.set("tab", filters.tab);
  if (filters.range !== "30d") params.set("range", filters.range);
  for (const key of ["environment", "category", "channel", "provider", "task_kind", "status", "q"] as const) {
    const value = filters[key];
    if (value) params.set(key, value);
  }
  if (filters.page !== 1) params.set("page", String(filters.page));
  if (filters.page_size !== 20) params.set("page_size", String(filters.page_size));
  history.replaceState(history.state, "", `${location.pathname}${params.size ? `?${params.toString()}` : ""}`);
}

export function operationsTimeRange(range: OperationsRange, now = new Date()) {
  const days = range === "7d" ? 7 : range === "90d" ? 90 : 30;
  return { from: new Date(now.getTime() - days * 86_400_000).toISOString(), to: now.toISOString() };
}
