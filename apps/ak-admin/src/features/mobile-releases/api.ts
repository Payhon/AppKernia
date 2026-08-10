import type { AdminMobileReleaseRequest } from "../../generated/api/types.gen";
import { toApiError } from "../../shared/api/error";
import { authSession } from "../auth/store";
import type { ManagedMobileRelease, MobileReleaseFilters, MobileReleaseInput, MobileReleasePage } from "./model";

async function data<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return ((await response.json()) as { data: T }).data;
}
const query = (filters: MobileReleaseFilters) => {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => { if (value !== "") params.set(key, String(value)); });
  return `?${params.toString()}`;
};
function request(input: MobileReleaseInput): AdminMobileReleaseRequest {
  return {
    package_type: input.package_type, platforms: input.platforms, version: input.version,
    minimum_native_version: input.minimum_native_version || null, titles: input.titles, contents: input.contents,
    package_file_id: input.source_type === "internal" ? input.package_file_id : null,
    external_url: input.source_type === "external" ? input.external_url || null : null,
    store_listing_ids: input.store_listing_ids, create_env: "upgrade_center", is_silently: input.is_silently,
    is_mandatory: input.is_mandatory, publish_now: input.publish_now,
    ...(input.lock_version === undefined ? {} : { lock_version: input.lock_version }),
  };
}
const json = (method: "POST" | "PATCH", body: unknown): RequestInit => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const root = (appId: string) => `/apps/${encodeURIComponent(appId)}/mobile/releases` as const;

export const mobileReleasesApi = {
  list: (appId: string, filters: MobileReleaseFilters) => data<MobileReleasePage>(`${root(appId)}${query(filters)}`),
  detail: (appId: string, id: string) => data<ManagedMobileRelease>(`${root(appId)}/${encodeURIComponent(id)}`),
  create: (appId: string, input: MobileReleaseInput) => data<ManagedMobileRelease>(root(appId), json("POST", request(input))),
  update: (appId: string, id: string, input: MobileReleaseInput) => data<ManagedMobileRelease>(`${root(appId)}/${encodeURIComponent(id)}`, json("PATCH", request(input))),
  action: (appId: string, id: string, action: "publish" | "unpublish", lockVersion: number) => data<ManagedMobileRelease>(`${root(appId)}/${encodeURIComponent(id)}/${action}`, json("POST", { lock_version: lockVersion })),
  delete: (appId: string, id: string) => data<{ deleted: boolean }>(`${root(appId)}/${encodeURIComponent(id)}`, { method: "DELETE" }),
  batchDelete: (appId: string, ids: string[]) => data<{ deleted_count: number }>(`${root(appId)}/batch-delete`, json("POST", { ids })),
};
