import type {
  AdminShareBinding,
  AdminShareBindingInput,
  AdminShareBindingListResponse,
  AdminShareBindingResponse,
  AdminShareConfig,
  AdminShareConfigInput,
  AdminShareConfigListResponse,
  AdminShareConfigResponse,
  AdminSharePreflight,
  AdminSharePreflightResponse,
} from "../../generated/api/types.gen";
import { toApiError } from "../../shared/api/error";
import { authSession } from "../auth/store";
import type { ShareConfigFilters } from "./model";

async function json<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return (await response.json()) as T;
}

function write(method: "POST" | "PATCH" | "PUT", body: object): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

export async function listShareConfigs(filters: ShareConfigFilters) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters))
    if (value !== undefined && value !== "") params.set(key, String(value));
  return (
    await json<AdminShareConfigListResponse>(
      `/share-configs?${params.toString()}`,
    )
  ).data;
}

export async function createShareConfig(input: AdminShareConfigInput) {
  return (
    await json<AdminShareConfigResponse>(
      "/share-configs",
      write("POST", input),
    )
  ).data;
}

export async function updateShareConfig(
  id: string,
  input: AdminShareConfigInput,
) {
  return (
    await json<AdminShareConfigResponse>(
      `/share-configs/${encodeURIComponent(id)}`,
      write("PATCH", input),
    )
  ).data;
}

export async function transitionShareConfig(
  id: string,
  transition: "activate" | "disable",
  lockVersion: number,
) {
  return (
    await json<AdminShareConfigResponse>(
      `/share-configs/${encodeURIComponent(id)}/${transition}`,
      write("POST", { lock_version: lockVersion }),
    )
  ).data;
}

export async function deleteShareConfig(id: string, lockVersion: number) {
  await json(
    `/share-configs/${encodeURIComponent(id)}?lock_version=${String(lockVersion)}`,
    { method: "DELETE" },
  );
}

export async function listAppShareBindings(
  appId: string,
): Promise<AdminShareBinding[]> {
  return (
    await json<AdminShareBindingListResponse>(
      `/apps/${encodeURIComponent(appId)}/share-bindings`,
    )
  ).data;
}

export async function preflightAppShareBinding(
  appId: string,
  provider: "wechat",
  input: AdminShareBindingInput,
): Promise<AdminSharePreflight> {
  return (
    await json<AdminSharePreflightResponse>(
      `/apps/${encodeURIComponent(appId)}/share-bindings/${provider}/preflight`,
      write("POST", input),
    )
  ).data;
}

export async function putAppShareBinding(
  appId: string,
  provider: "wechat",
  input: AdminShareBindingInput,
): Promise<AdminShareBinding> {
  return (
    await json<AdminShareBindingResponse>(
      `/apps/${encodeURIComponent(appId)}/share-bindings/${provider}`,
      write("PUT", input),
    )
  ).data;
}

export async function deleteAppShareBinding(
  appId: string,
  provider: "wechat",
  lockVersion: number,
) {
  await json(
    `/apps/${encodeURIComponent(appId)}/share-bindings/${provider}?lock_version=${String(lockVersion)}`,
    { method: "DELETE" },
  );
}

export type { AdminShareConfig };
