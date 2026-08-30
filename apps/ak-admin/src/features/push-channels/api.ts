import type {
  AdminPushDeliverySummaryResponse,
  AdminPushProviderCatalogResponse,
  AdminPushProviderConfigListResponse,
  AdminPushProviderConfigRequest,
  AdminPushProviderConfigResponse,
  AdminPushSecretRotationRequestWritable,
  AdminPushTestDeviceListResponse,
  AdminPushTestRequest,
  AdminPushTestResponse,
  PushEnvironment,
  PushWritableProvider,
} from "../../generated/api/types.gen";
import { toApiError } from "../../shared/api/error";
import { authSession } from "../auth/store";

async function json<T>(path: `/${string}`, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  return (await response.json()) as T;
}

function write(method: "POST" | "PUT", body: object): RequestInit {
  return { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
}

const appPath = (appId: string) => `/apps/${encodeURIComponent(appId)}` as const;

export async function listPushProviderCatalog(appId: string) {
  return (await json<AdminPushProviderCatalogResponse>(`${appPath(appId)}/push-provider-catalog`)).data.items;
}

export async function listPushProviderConfigs(appId: string, environment: PushEnvironment) {
  return (await json<AdminPushProviderConfigListResponse>(`${appPath(appId)}/push-provider-configs?environment=${encodeURIComponent(environment)}`)).data.items;
}

export async function upsertPushProviderConfig(appId: string, provider: PushWritableProvider, input: AdminPushProviderConfigRequest) {
  return (await json<AdminPushProviderConfigResponse>(`${appPath(appId)}/push-provider-configs/${provider}`, write("PUT", input))).data;
}

export async function rotatePushProviderSecret(appId: string, id: string, input: AdminPushSecretRotationRequestWritable) {
  return (await json<AdminPushProviderConfigResponse>(`${appPath(appId)}/push-provider-configs/${encodeURIComponent(id)}/rotate-secret`, write("POST", input))).data;
}

export async function transitionPushProviderConfig(appId: string, id: string, action: "preflight" | "activate" | "disable", lockVersion: number) {
  return (await json<AdminPushProviderConfigResponse>(`${appPath(appId)}/push-provider-configs/${encodeURIComponent(id)}/${action}`, write("POST", { lock_version: lockVersion }))).data;
}

export async function listPushTestDevices(appId: string, provider: PushWritableProvider) {
  return (await json<AdminPushTestDeviceListResponse>(`${appPath(appId)}/push-devices?provider=${provider}`)).data.items;
}

export async function sendPushTest(appId: string, configId: string, input: AdminPushTestRequest) {
  return (await json<AdminPushTestResponse>(`${appPath(appId)}/push-provider-configs/${encodeURIComponent(configId)}/test`, write("POST", input))).data;
}

export async function getPushDeliverySummary(appId: string) {
  return (await json<AdminPushDeliverySummaryResponse>(`${appPath(appId)}/push-delivery-summary`)).data;
}
