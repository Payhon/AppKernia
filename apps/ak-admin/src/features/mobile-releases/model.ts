import { z } from "zod";

export const mobileReleasePlatforms = ["android", "ios", "harmony"] as const;
export type MobileReleasePlatform = (typeof mobileReleasePlatforms)[number];

const semverPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

interface ParsedSemver {
  core: [number, number, number];
}

function parseSemver(value: string): ParsedSemver | null {
  const match = semverPattern.exec(value);
  if (!match) return null;
  return {
    core: [Number(match[1]), Number(match[2]), Number(match[3])],
  };
}

export function compareSemver(left: string, right: string): number {
  const a = parseSemver(left);
  const b = parseSemver(right);
  if (!a || !b) throw new Error("INVALID_SEMVER");
  for (let index = 0; index < a.core.length; index += 1) {
    const difference = (a.core[index] ?? 0) - (b.core[index] ?? 0);
    if (difference !== 0) return Math.sign(difference);
  }
  return 0;
}

const httpsUrl = z.union([z.literal(""), z.url().refine((value) => value.startsWith("https://"))]);

export const mobileReleaseInputSchema = z.object({
  platform: z.enum(mobileReleasePlatforms),
  current_version: z.string().trim().max(64).regex(semverPattern),
  minimum_version: z.string().trim().max(64).regex(semverPattern),
  upgrade_url: httpsUrl,
  active: z.boolean(),
  release_notes: z.object({
    "zh-CN": z.string().trim().min(1).max(10_000),
    "en-US": z.string().trim().min(1).max(10_000),
  }),
  lock_version: z.number().int().positive().optional(),
}).superRefine((value, context) => {
  if (parseSemver(value.minimum_version) && parseSemver(value.current_version) && compareSemver(value.minimum_version, value.current_version) > 0) {
    context.addIssue({ code: "custom", path: ["minimum_version"], message: "minimum_newer" });
  }
  if (value.active && value.upgrade_url === "") {
    context.addIssue({ code: "custom", path: ["upgrade_url"], message: "active_url_required" });
  }
});

export type MobileReleaseInput = z.infer<typeof mobileReleaseInputSchema>;

export function defaultMobileReleaseInput(): MobileReleaseInput {
  return {
    platform: "android",
    current_version: "1.0.0",
    minimum_version: "1.0.0",
    upgrade_url: "",
    active: false,
    release_notes: { "zh-CN": "", "en-US": "" },
  };
}
