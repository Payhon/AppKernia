import { z } from "zod";
import type {
  AdminJobSchedule,
  AdminJobScheduleRequest,
} from "../../generated/api/types.gen";

export interface ScheduleValues {
  code: string;
  name: string;
  handler_key: string;
  cron_expression: string;
  time_zone: string;
  payload: string;
  overlap_policy: AdminJobScheduleRequest["overlap_policy"];
  misfire_policy: AdminJobScheduleRequest["misfire_policy"];
  timeout_seconds: number;
  max_attempts: number;
}

export const emptyScheduleValues: ScheduleValues = {
  code: "",
  name: "",
  handler_key: "system.health.snapshot",
  cron_expression: "0 * * * *",
  time_zone: "UTC",
  payload: "{}",
  overlap_policy: "skip",
  misfire_policy: "fire_once",
  timeout_seconds: 300,
  max_attempts: 3,
};

const jsonObject = z.string().refine((raw) => {
  try {
    const value: unknown = JSON.parse(raw);
    return typeof value === "object" && value !== null && !Array.isArray(value);
  } catch {
    return false;
  }
});

export const scheduleValuesSchema = z.object({
  code: z.string().trim().regex(/^[a-z][a-z0-9_.-]{0,95}$/),
  name: z.string().trim().min(1).max(160),
  handler_key: z.string().trim().min(1).max(160),
  cron_expression: z
    .string()
    .trim()
    .max(128)
    .refine((value) => value.split(/\s+/).length === 5),
  time_zone: z.string().trim().min(1).max(64),
  payload: jsonObject,
  overlap_policy: z.enum(["allow", "skip", "replace"]),
  misfire_policy: z.enum(["ignore", "fire_once", "catch_up"]),
  timeout_seconds: z.coerce.number().int().min(1).max(86400),
  max_attempts: z.coerce.number().int().min(1).max(100),
});

export function toScheduleRequest(
  values: ScheduleValues,
): AdminJobScheduleRequest {
  const parsed = scheduleValuesSchema.parse(values);
  return {
    code: parsed.code.trim(),
    name: parsed.name.trim(),
    handler_key: parsed.handler_key.trim(),
    cron_expression: parsed.cron_expression.trim().replace(/\s+/g, " "),
    time_zone: parsed.time_zone.trim(),
    payload: JSON.parse(parsed.payload) as Record<string, unknown>,
    queue_name: "default",
    overlap_policy: parsed.overlap_policy,
    misfire_policy: parsed.misfire_policy,
    timeout_seconds: parsed.timeout_seconds,
    max_attempts: parsed.max_attempts,
  };
}

export function fromSchedule(value: AdminJobSchedule): ScheduleValues {
  return {
    code: value.code,
    name: value.name,
    handler_key: value.handler_key,
    cron_expression: value.cron_expression,
    time_zone: value.time_zone,
    payload: JSON.stringify(value.payload, null, 2),
    overlap_policy: value.overlap_policy,
    misfire_policy: value.misfire_policy,
    timeout_seconds: value.timeout_seconds,
    max_attempts: value.max_attempts,
  };
}

export function cronDescriptionKey(expression: string): string {
  const normalized = expression.trim().replace(/\s+/g, " ");
  if (normalized === "* * * * *") return "schedules.cron.every_minute";
  if (normalized === "0 * * * *") return "schedules.cron.hourly";
  if (normalized === "0 0 * * *") return "schedules.cron.daily";
  return "schedules.cron.custom";
}
