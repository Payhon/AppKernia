import { z } from "zod";

import { toApiError } from "../../shared/api/error";
import { authSession } from "../auth/store";
import {
  appLoginProviderBindingsSchema,
  loginProviderCatalogItemSchema,
  loginProviderConfigPageSchema,
  loginProviderConfigSchema,
  type AppLoginProviderBindingsWriteInput,
  type LoginProviderConfigFilters,
  type LoginProviderConfigWriteInput,
  type LoginProviderSecretRotationInput,
} from "./model";

const catalogResponseSchema = z.object({ data: z.object({ items: z.array(loginProviderCatalogItemSchema) }) });
const configPageResponseSchema = z.object({ data: loginProviderConfigPageSchema });
const configResponseSchema = z.object({ data: loginProviderConfigSchema });
const bindingsResponseSchema = z.object({ data: appLoginProviderBindingsSchema });

async function request<T>(path: `/${string}`, schema: z.ZodType<T>, init?: RequestInit): Promise<T> {
  const response = await authSession.adminRequest(path, init);
  if (!response.ok) throw await toApiError(response);
  const body: unknown = await response.json();
  return schema.parse(body);
}

function write(method: "POST" | "PATCH" | "PUT", body: object): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

export async function listLoginProviderCatalog() {
  return (await request("/login-provider-catalog", catalogResponseSchema)).data.items;
}

export async function listLoginProviderConfigs(filters: LoginProviderConfigFilters) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== "") params.set(key, String(value));
  }
  return (await request(`/login-provider-configs?${params.toString()}`, configPageResponseSchema)).data;
}

export async function getLoginProviderConfig(id: string) {
  return (await request(`/login-provider-configs/${encodeURIComponent(id)}`, configResponseSchema)).data;
}

export async function createLoginProviderConfig(input: LoginProviderConfigWriteInput) {
  const createInput = {
    name: input.name,
    description: input.description,
    provider_code: input.provider_code,
    external_client_id: input.external_client_id,
    config_schema_version: input.config_schema_version,
    public_config: input.public_config,
  };
  return (await request("/login-provider-configs", configResponseSchema, write("POST", createInput))).data;
}

export async function updateLoginProviderConfig(id: string, input: LoginProviderConfigWriteInput) {
  return (await request(`/login-provider-configs/${encodeURIComponent(id)}`, configResponseSchema, write("PATCH", input))).data;
}

export async function rotateLoginProviderSecret(id: string, input: LoginProviderSecretRotationInput) {
  return (await request(`/login-provider-configs/${encodeURIComponent(id)}/rotate-secret`, configResponseSchema, write("POST", input))).data;
}

export async function transitionLoginProviderConfig(
  id: string,
  transition: "preflight" | "activate" | "disable",
  lockVersion: number,
) {
  return (await request(`/login-provider-configs/${encodeURIComponent(id)}/${transition}`, configResponseSchema, write("POST", { lock_version: lockVersion }))).data;
}

export async function deleteLoginProviderConfig(id: string, lockVersion: number): Promise<void> {
  const response = await authSession.adminRequest(
    `/login-provider-configs/${encodeURIComponent(id)}?lock_version=${String(lockVersion)}`,
    { method: "DELETE" },
  );
  if (!response.ok) throw await toApiError(response);
}

export async function listAppLoginProviderBindings(appId: string) {
  return (await request(`/apps/${encodeURIComponent(appId)}/login-provider-bindings`, bindingsResponseSchema)).data.items;
}

export async function putAppLoginProviderBindings(appId: string, input: AppLoginProviderBindingsWriteInput) {
  return (await request(
    `/apps/${encodeURIComponent(appId)}/login-provider-bindings`,
    bindingsResponseSchema,
    write("PUT", input),
  )).data.items;
}
