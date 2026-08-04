import { describe, expect, it } from "vitest";
import {
  cronDescriptionKey,
  emptyScheduleValues,
  scheduleValuesSchema,
  toScheduleRequest,
} from "./schema";

describe("scheduled jobs form", () => {
  it("normalizes a safe request", () => {
    const request = toScheduleRequest({
      ...emptyScheduleValues,
      code: " health.snapshot ",
      name: " Health snapshot ",
      cron_expression: "0   * * * *",
    });
    expect(request).toMatchObject({
      code: "health.snapshot",
      name: "Health snapshot",
      cron_expression: "0 * * * *",
      queue_name: "default",
      payload: {},
    });
  });

  it("rejects executable-looking non-object payloads and malformed cron", () => {
    expect(
      scheduleValuesSchema.safeParse({
        ...emptyScheduleValues,
        payload: '"rm -rf /"',
      }).success,
    ).toBe(false);
    expect(
      scheduleValuesSchema.safeParse({
        ...emptyScheduleValues,
        cron_expression: "@every 5m",
      }).success,
    ).toBe(false);
  });

  it("uses semantic cron descriptions", () => {
    expect(cronDescriptionKey("0 * * * *")).toBe("schedules.cron.hourly");
    expect(cronDescriptionKey("15 9 * * 1-5")).toBe(
      "schedules.cron.custom",
    );
  });
});
