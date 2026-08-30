import { beforeEach, describe, expect, it, vi } from "vitest";
import { authSession } from "../auth/store";
import { getOperationsSummary, retryNotificationTasks } from "./api";
import { readOperationsFilters } from "./model";

describe("notification operations API", () => {
  const request = vi.spyOn(authSession, "adminRequest");
  beforeEach(() => {
    request.mockReset();
    request.mockResolvedValue(new Response(JSON.stringify({ data: { items: [] } }), { status: 200, headers: { "Content-Type": "application/json" } }));
  });

  it("always scopes reads to the selected App", async () => {
    await getOperationsSummary("018f08d0-3b00-7000-8000-000000000001", readOperationsFilters("?range=7d"));
    expect(request.mock.calls[0]?.[0]).toMatch(/^\/apps\/018f08d0-3b00-7000-8000-000000000001\/notification-operations\/summary\?/);
  });

  it("sends only task ids and explicit duplicate-risk acknowledgement", async () => {
    await retryNotificationTasks("018f08d0-3b00-7000-8000-000000000001", { items: [{ task_id: "018f08d0-3b00-7000-8000-000000000002" }], acknowledge_duplicate_risk: true });
    expect(request.mock.calls[0]?.[1]?.body).toBe(JSON.stringify({ items: [{ task_id: "018f08d0-3b00-7000-8000-000000000002" }], acknowledge_duplicate_risk: true }));
  });
});
