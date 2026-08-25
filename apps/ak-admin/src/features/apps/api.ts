import { authSession } from "../auth/store";
import { toApiError } from "../../shared/api/error";
import type { AppListFilters, AppMember, AppMemberCreateInput, AppMemberUpdateInput, ApplicationInput, ManagedApplication, ManagedPage, AppPageInput, MemberListFilters, PageListFilters, Paginated } from "./model";

async function data<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return ((await response.json()) as { data: T }).data;
}
function query(filters: object) {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => { if (value !== "" && value !== undefined && value !== null) params.set(key, String(value)); });
  return params.size ? `?${params.toString()}` : "";
}
const json = (method: "POST" | "PATCH", body: unknown): RequestInit => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const appPath = (appId: string, suffix = "") => `/apps/${encodeURIComponent(appId)}${suffix}` as const;

export const appAdminApi = {
  list: (filters: AppListFilters) => data<Paginated<ManagedApplication>>(`/apps${query(filters)}`),
  create: (input: ApplicationInput) => data<ManagedApplication>("/apps", json("POST", input)),
  update: (id: string, input: ApplicationInput) => data<ManagedApplication>(`/apps/${encodeURIComponent(id)}`, json("PATCH", input)),
  delete: (id: string) => data<{ deleted: boolean }>(`/apps/${encodeURIComponent(id)}`, { method: "DELETE" }),
  batchDelete: (ids: string[]) => data<{ deleted_count: number }>("/apps/batch-delete", json("POST", { ids })),
  setStatus: (id: string, action: "enable" | "disable", lockVersion: number) => data<ManagedApplication>(`/apps/${encodeURIComponent(id)}/${action}`, json("POST", { lock_version: lockVersion })),
  publishOnboarding: (id: string, expectedPublishedVersion: number) => data<ManagedApplication>(appPath(id, "/startup/onboarding/publish"), json("POST", { expected_published_version: expectedPublishedVersion })),
  members: (appId: string, filters: MemberListFilters) => data<Paginated<AppMember>>(appPath(appId, `/users${query(filters)}`)),
  createMember: (appId: string, input: AppMemberCreateInput) => data<AppMember>(appPath(appId, "/users"), json("POST", input)),
  updateMember: (appId: string, memberId: string, input: AppMemberUpdateInput) => data<AppMember>(appPath(appId, `/users/${encodeURIComponent(memberId)}`), json("PATCH", input)),
  memberAction: (appId: string, memberId: string, action: "enable" | "disable" | "unlock" | "revoke-sessions", lockVersion: number) => data<AppMember>(appPath(appId, `/users/${encodeURIComponent(memberId)}${action === "revoke-sessions" ? "/sessions/revoke" : `/${action}`}`), json("POST", { lock_version: lockVersion })),
  resetMemberPassword: (appId: string, memberId: string, newPassword: string, lockVersion: number) => data<AppMember>(appPath(appId, `/users/${encodeURIComponent(memberId)}/reset-password`), json("POST", { new_password: newPassword, lock_version: lockVersion })),
  pages: (appId: string, filters: PageListFilters) => data<Paginated<ManagedPage>>(appPath(appId, `/content/pages${query(filters)}`)),
  createPage: (appId: string, input: AppPageInput) => data<ManagedPage>(appPath(appId, "/content/pages"), json("POST", input)),
  updatePage: (appId: string, slug: string, input: AppPageInput) => data<ManagedPage>(appPath(appId, `/content/pages/${encodeURIComponent(slug)}`), json("PATCH", input)),
  publishPage: (appId: string, slug: string, lockVersion: number) => data<ManagedPage>(appPath(appId, `/content/pages/${encodeURIComponent(slug)}/publish`), json("POST", { lock_version: lockVersion })),
  deletePage: (appId: string, slug: string, lockVersion: number) => data<undefined>(appPath(appId, `/content/pages/${encodeURIComponent(slug)}${query({ lock_version: lockVersion })}`), { method: "DELETE" }),
};
