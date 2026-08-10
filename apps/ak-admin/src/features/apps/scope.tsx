import { useRouterState } from "@tanstack/react-router";
import { useAuthStore } from "../auth/store";
import { useManagedApplications } from "./hooks";
import { isUUID, useAppSelectionStore } from "./selection-store";

export interface AppScope { appId: string | null; appName: string | null; disabled: boolean; }

const appScopedRoutes = new Set(["/app/users", "/app/upgrade-center", "/system/mobile/releases"]);

export function readAppId(search: string): string | null {
  const value = new URLSearchParams(search).get("app_id");
  return value && isUUID(value) ? value : null;
}

export function resolveAppId(urlAppId: string | null, rememberedAppId: string | null): string | null {
  return urlAppId ?? rememberedAppId;
}

export function requiresAppSelection(pathname: string): boolean {
  const normalized = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  return appScopedRoutes.has(normalized) || normalized.startsWith("/app/content/");
}

export function withSelectedApp(search: Record<string, unknown>, appId: string | undefined): Record<string, unknown> {
  return {
    ...search,
    app_id: appId ?? undefined,
    ...(Object.hasOwn(search, "page") ? { page: 1 } : {}),
  };
}

export function useAppScope(): AppScope {
  const search = useRouterState({ select: (state) => state.location.searchStr });
  const tenantId = useAuthStore((state) => state.context?.active_tenant.id ?? null);
  const rememberedAppId = useAppSelectionStore((state) => tenantId ? state.appIdByTenant[tenantId] ?? null : null);
  const apps = useManagedApplications({ q: "", page: 1, page_size: 100 });
  const appId = resolveAppId(readAppId(search), rememberedAppId);
  const application = apps.data?.items.find((item) => item.id === appId) ?? null;
  return { appId, appName: application?.name ?? null, disabled: application?.status === "disabled" };
}

export function AppScopeContext({ children }: { children: (scope: AppScope) => React.ReactNode }) {
  const scope = useAppScope();
  return <>{children(scope)}</>;
}
