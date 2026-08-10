import { create } from "zustand";

export interface AppSelectionStorage {
  getItem: (key: string) => string | null;
  removeItem: (key: string) => void;
  setItem: (key: string, value: string) => void;
}

export type AppSelectionsByTenant = Record<string, string>;

interface AppSelectionState {
  appIdByTenant: AppSelectionsByTenant;
  setSelection: (tenantId: string, appId: string | null) => void;
}

export const appSelectionStorageKey = "ak.admin.app-selection.v1";

export function isUUID(value: string): boolean {
  return /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(value);
}

function browserStorage(): AppSelectionStorage | undefined {
  try {
    return typeof window === "undefined" ? undefined : window.localStorage;
  } catch {
    return undefined;
  }
}

export function readAppSelections(storage: AppSelectionStorage | null | undefined = browserStorage()): AppSelectionsByTenant {
  try {
    const raw = storage?.getItem(appSelectionStorageKey);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return {};
    return Object.fromEntries(
      Object.entries(parsed).filter(([tenantId, appId]) => isUUID(tenantId) && typeof appId === "string" && isUUID(appId)),
    );
  } catch {
    return {};
  }
}

export function writeAppSelections(
  selections: AppSelectionsByTenant,
  storage: AppSelectionStorage | null | undefined = browserStorage(),
) {
  try {
    if (Object.keys(selections).length === 0) storage?.removeItem(appSelectionStorageKey);
    else storage?.setItem(appSelectionStorageKey, JSON.stringify(selections));
  } catch {
    // A blocked or full localStorage must not make App-scoped navigation unusable.
  }
}

export function updateAppSelection(selections: AppSelectionsByTenant, tenantId: string, appId: string | null): AppSelectionsByTenant {
  if (!isUUID(tenantId) || (appId !== null && !isUUID(appId))) return selections;
  if (appId === null) {
    return Object.fromEntries(Object.entries(selections).filter(([storedTenantId]) => storedTenantId !== tenantId));
  }
  return { ...selections, [tenantId]: appId };
}

export const useAppSelectionStore = create<AppSelectionState>((set) => ({
  appIdByTenant: readAppSelections(),
  setSelection: (tenantId, appId) => {
    set((state) => {
      const appIdByTenant = updateAppSelection(state.appIdByTenant, tenantId, appId);
      writeAppSelections(appIdByTenant);
      return { appIdByTenant };
    });
  },
}));
