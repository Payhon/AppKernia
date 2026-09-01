import { z } from "zod";
import type { AdminPublicWebConfig, AdminPublicWebConfigRequest } from "../../generated/api/types.gen";
import { authSession } from "../auth/store";
import { toApiError } from "../../shared/api/error";

const translation = z.object({
  name: z.string().trim().max(160),
  introduction: z.string().trim().max(20_000),
  promotion_title: z.string().trim().max(160),
  promotion_description: z.string().trim().max(500),
  promotion_button_label: z.string().trim().max(80),
});
export const publicWebSchema = z.object({
  enabled: z.boolean(), apk_enabled: z.boolean(), promotion_enabled: z.boolean(), lock_version: z.number().int().min(0),
  translations: z.object({ "zh-CN": translation, "en-US": translation }),
  stores: z.array(z.object({ id: z.uuid(), name: z.string(), enabled: z.boolean(), priority: z.number().int(), platform: z.enum(["", "android", "ios", "harmony"]), web_url: z.string().max(2048) })).max(100),
}).superRefine((value, ctx) => {
  for (const locale of ["zh-CN", "en-US"] as const) {
    if (value.enabled && (!value.translations[locale].name || !value.translations[locale].introduction)) ctx.addIssue({ code: "custom", path: ["translations", locale], message: "apps.public_web.validation" });
  }
  const ids = new Set<string>();
  value.stores.forEach((store, index) => {
    if (ids.has(store.id)) ctx.addIssue({ code: "custom", path: ["stores", index], message: "apps.public_web.validation" });
    ids.add(store.id);
    if (!store.web_url) return;
    try { const url = new URL(store.web_url); if (url.protocol !== "https:" || url.username || url.password || /\s/.test(store.web_url) || !store.platform) throw new Error("invalid"); }
    catch { ctx.addIssue({ code: "custom", path: ["stores", index, "web_url"], message: "apps.public_web.validation" }); }
  });
});
export type PublicWebFormValues = z.infer<typeof publicWebSchema>;
export async function publicWebConfig(appId: string, input?: AdminPublicWebConfigRequest): Promise<AdminPublicWebConfig> {
  const response = await authSession.adminRequest(`/apps/${encodeURIComponent(appId)}/public-web-config`, input ? { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) } : undefined);
  if (!response.ok) throw await toApiError(response);
  return ((await response.json()) as { data: AdminPublicWebConfig }).data;
}
