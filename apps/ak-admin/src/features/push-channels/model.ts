import { z } from "zod";
import type { AdminPushProviderCatalogItem, AdminPushProviderConfig, PushWritableProvider } from "../../generated/api/types.gen";

export const pushProviders: PushWritableProvider[] = ["apns", "fcm", "huawei_android", "honor", "xiaomi", "oppo", "vivo", "meizu", "harmony"];
export interface PushConfigEditor { catalog: AdminPushProviderCatalogItem; config: AdminPushProviderConfig | null }

export const fieldValuesSchema = (fields: string[], maxLength = 2048) => z.record(z.string(), z.string().trim().min(1).max(maxLength)).refine((value) => fields.every((field) => Boolean(value[field]?.trim())));
export const testPushSchema = z.object({ push_device_id: z.uuid(), title: z.string().trim().min(1).max(64), body: z.string().trim().min(1).max(180) });

export function providerState(config: AdminPushProviderConfig | null): "unconfigured" | "draft" | "ready" | "active" | "faulted" | "disabled" {
  if (!config) return "unconfigured";
  if (config.status === "active") return "active";
  if (config.status === "faulted") return "faulted";
  if (config.status === "disabled") return "disabled";
  return config.last_preflight_status === "ready" ? "ready" : "draft";
}
