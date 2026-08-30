import { describe, expect, it, vi } from "vitest";

import type { ManagedApplication } from "./model";
import { createApplicationActionItems, type ApplicationActionHandlers, type ApplicationActionLabels } from "./application-actions";

const labels: ApplicationActionLabels = {
  edit: "Edit",
  upgradeCenter: "Upgrade center",
  content: "Content",
  clientConfig: "Client configuration",
  enable: "Enable",
  disable: "Disable",
  delete: "Delete",
};
const handlers: ApplicationActionHandlers = {
  edit: vi.fn(),
  upgradeCenter: vi.fn(),
  content: vi.fn(),
  clientConfig: vi.fn(),
  changeStatus: vi.fn(),
  delete: vi.fn(),
};
const application = {
  id: "00000000-0000-4000-8000-000000000001",
  status: "disabled",
  is_default: false,
} as ManagedApplication;

describe("application action menu", () => {
  it("uses one icon for every available action and preserves the expected order", () => {
    const permissions = new Set([
      "app.application.update",
      "app.share_binding.read",
      "app.scanner_config.read",
      "app.application.disable",
      "app.application.delete",
    ]);
    const items = createApplicationActionItems(application, permissions, labels, handlers, false);
    expect(items.map((item) => item?.key)).toEqual(["edit", "upgrade-center", "content", "client-config", "status", "delete"]);
    expect(items.every((item) => item !== null && "icon" in item && item.icon !== undefined)).toBe(true);
  });

  it("shows client configuration when only scanner configuration is readable", () => {
    const items = createApplicationActionItems(application, new Set(["app.scanner_config.read"]), labels, handlers, false);
    expect(items.map((item) => item?.key)).toEqual(["upgrade-center", "content", "client-config"]);
  });

  it("hides unauthorized and ineligible actions without hiding safe destinations", () => {
    const activeDefault = { ...application, status: "active", is_default: true } as ManagedApplication;
    const items = createApplicationActionItems(activeDefault, new Set(), labels, handlers, false);
    expect(items.map((item) => item?.key)).toEqual(["upgrade-center", "content"]);
  });
});
