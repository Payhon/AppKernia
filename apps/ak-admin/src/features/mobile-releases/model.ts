import { z } from "zod";
import type { AdminMobileRelease } from "../../generated/api/types.gen";

export const mobileReleasePlatforms = ["android", "ios", "harmony"] as const;
export const mobileReleasePackageTypes = ["native_app", "wgt"] as const;
export const mobileReleasePublishStatuses = ["draft", "online", "partial", "offline"] as const;
export type MobileReleasePlatform = (typeof mobileReleasePlatforms)[number];
export type ManagedMobileRelease = AdminMobileRelease;
export type MobileApplicationType = "uni_app" | "uni_app_x";

const semverPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

interface ParsedSemver { core: [number, number, number]; }
function parseSemver(value: string): ParsedSemver | null {
  const match = semverPattern.exec(value);
  return match ? { core: [Number(match[1]), Number(match[2]), Number(match[3])] } : null;
}
export function compareSemver(left: string, right: string): number {
  const a = parseSemver(left); const b = parseSemver(right);
  if (!a || !b) throw new Error("INVALID_SEMVER");
  for (let index = 0; index < a.core.length; index += 1) {
    const difference = (a.core[index] ?? 0) - (b.core[index] ?? 0);
    if (difference !== 0) return Math.sign(difference);
  }
  return 0;
}

export const mobileReleaseInputSchema = z.object({
  package_type: z.enum(mobileReleasePackageTypes),
  platforms: z.array(z.enum(mobileReleasePlatforms)).min(1).max(3),
  version: z.string().trim().max(64).regex(semverPattern, { message: "semver" }),
  minimum_native_version: z.string().trim().max(64),
  titles: z.object({ "zh-CN": z.string().trim().max(200), "en-US": z.string().trim().max(200) }),
  contents: z.object({ "zh-CN": z.string().trim().max(10_000), "en-US": z.string().trim().max(10_000) }),
  source_type: z.enum(["internal", "external"]),
  package_file_id: z.uuid().nullable(),
  external_url: z.string().trim().max(2000),
  store_listing_ids: z.array(z.uuid()).max(100),
  is_silently: z.boolean(),
  is_mandatory: z.boolean(),
  publish_now: z.boolean(),
  lock_version: z.number().int().positive().optional(),
}).superRefine((value, context) => {
  if (value.package_type === "native_app" && value.platforms.length !== 1) context.addIssue({ code: "custom", path: ["platforms"], message: "native_single_platform" });
  if (value.package_type === "wgt" && !semverPattern.test(value.minimum_native_version)) context.addIssue({ code: "custom", path: ["minimum_native_version"], message: "semver" });
  if (value.minimum_native_version && semverPattern.test(value.minimum_native_version) && semverPattern.test(value.version) && compareSemver(value.minimum_native_version, value.version) > 0) context.addIssue({ code: "custom", path: ["minimum_native_version"], message: "minimum_newer" });
  if (value.source_type === "internal" && value.publish_now && !value.package_file_id) context.addIssue({ code: "custom", path: ["package_file_id"], message: "source_required" });
  if (value.source_type === "external" && value.publish_now && !value.external_url) context.addIssue({ code: "custom", path: ["external_url"], message: "source_required" });
  if (value.source_type === "external" && value.external_url && !value.external_url.startsWith("https://")) context.addIssue({ code: "custom", path: ["external_url"], message: "https" });
  if (value.publish_now) {
    (["zh-CN", "en-US"] as const).forEach((locale) => {
      if (!value.titles[locale]) context.addIssue({ code: "custom", path: ["titles", locale], message: "bilingual_required" });
      if (!value.contents[locale]) context.addIssue({ code: "custom", path: ["contents", locale], message: "bilingual_required" });
    });
  }
});

export function mobileReleaseInputSchemaFor(appType: MobileApplicationType) {
  return mobileReleaseInputSchema.superRefine((value, context) => {
    if (appType === "uni_app_x" && value.package_type === "wgt") {
      context.addIssue({ code: "custom", path: ["package_type"], message: "unsupported_package_type" });
    }
    if (value.package_type === "native_app" && value.source_type === "internal" && value.platforms[0] !== "android") {
      context.addIssue({ code: "custom", path: ["source_type"], message: "unsupported_delivery_mode" });
    }
  });
}

export type MobileReleaseInput = z.infer<typeof mobileReleaseInputSchema>;
export interface MobileReleaseFilters { q: string; package_type: string; platform: string; publish_status: string; page: number; page_size: number; }
export interface MobileReleasePage { items: ManagedMobileRelease[]; page: number; page_size: number; total: number; }

export function defaultMobileReleaseInput(packageType: "native_app" | "wgt" = "native_app"): MobileReleaseInput {
  return { package_type: packageType, platforms: ["android"], version: "1.0.0", minimum_native_version: packageType === "wgt" ? "1.0.0" : "", titles: { "zh-CN": "", "en-US": "" }, contents: { "zh-CN": "", "en-US": "" }, source_type: "external", package_file_id: null, external_url: "", store_listing_ids: [], is_silently: false, is_mandatory: false, publish_now: false };
}

export function releaseInput(item: ManagedMobileRelease): MobileReleaseInput {
  return { package_type: item.package_type, platforms: item.platforms, version: item.version, minimum_native_version: item.minimum_native_version ?? "", titles: item.titles, contents: item.contents, source_type: item.package_file_id ? "internal" : "external", package_file_id: item.package_file_id ?? null, external_url: item.external_url ?? "", store_listing_ids: item.store_listing_ids, is_silently: item.is_silently, is_mandatory: item.is_mandatory, publish_now: false, lock_version: item.lock_version };
}

export function releaseCapabilityError(appType: MobileApplicationType, item: Pick<ManagedMobileRelease, "package_type" | "platforms" | "package_file_id">): "unsupported_package_type" | "unsupported_delivery_mode" | null {
  if (appType === "uni_app_x" && item.package_type === "wgt") return "unsupported_package_type";
  if (item.package_type === "native_app" && item.package_file_id && item.platforms[0] !== "android") return "unsupported_delivery_mode";
  return null;
}
