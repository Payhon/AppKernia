import type { AdminMenuItem } from "../generated/api/types.gen";
import { generatedRouteRegistry } from "../generated/route-registry";

export type RegisteredRoute = (typeof generatedRouteRegistry)[number];

const routeByComponentKey = new Map<string, RegisteredRoute>(
  generatedRouteRegistry.map((route) => [route.componentKey, route]),
);
const implementedComponentKeys = new Set([
  "dashboard",
  "profile.basic",
  "profile.security",
  "profile.connections",
  "auth.oauth-callback",
  "system.settings.configs",
  "system.settings.dictionaries",
  "system.settings.regions",
  "system.settings.modules",
  "system.storage.files",
  "system.notifications.notices",
  "system.notifications.messages",
  "system.notifications.templates",
  "system.notifications.deliveries",
  "system.integrations.schedules",
  "system.integrations.api-clients",
  "system.integrations.webhooks",
  "system.integrations.webhook-deliveries",
  "system.users.accounts",
  "system.users.departments",
  "system.users.positions",
  "system.users.tenants",
  "system.access.roles",
  "system.access.permissions",
  "system.access.menus",
  "system.security.operation-logs",
  "system.security.login-logs",
  "system.security.events",
	"system.security.block-rules",
  "system.security.event-detail",
  "system.monitoring.sessions",
	"system.monitoring.health",
]);

export interface ResolvedMenuItem {
  code: string;
  i18nKey: string;
  path: string;
  componentKey: string;
}

export function findRegisteredRoute(
  componentKey: string,
): RegisteredRoute | undefined {
  return routeByComponentKey.get(componentKey);
}

export function isImplementedRoute(componentKey: string): boolean {
  return implementedComponentKeys.has(componentKey);
}

export function canAccessRoute(
  route: RegisteredRoute,
  permissions: ReadonlySet<string>,
  featureFlags: Readonly<Record<string, boolean>>,
): boolean {
  const featureAllowed =
    route.featureFlag === null || featureFlags[route.featureFlag] === true;
  return (
    featureAllowed &&
    route.permissions.every((permission) => permissions.has(permission))
  );
}

export function resolveBackendMenus(
  menus: readonly AdminMenuItem[],
  permissions: ReadonlySet<string>,
  featureFlags: Readonly<Record<string, boolean>>,
  onUnknown: (componentKey: string) => void = () => undefined,
): ResolvedMenuItem[] {
  return menus.flatMap((menu) => {
    if (menu.type !== "page" || !menu.component_key) return [];
    const route = findRegisteredRoute(menu.component_key);
    if (!route) {
      onUnknown(menu.component_key);
      return [];
    }
    if (!canAccessRoute(route, permissions, featureFlags)) return [];
    return [
      {
        code: menu.code,
        componentKey: route.componentKey,
        i18nKey: menu.i18n_key,
        path: route.path,
      },
    ];
  });
}

export function isSafeInternalRedirect(
  value: string | null,
): value is
  | "/dashboard"
  | "/profile/basic"
  | "/profile/security"
  | "/profile/connections"
  | "/system/settings/configs"
  | "/system/settings/dictionaries"
  | "/system/settings/regions"
  | "/system/settings/modules"
  | "/system/storage/files"
  | "/system/users/accounts"
  | "/system/users/departments"
  | "/system/users/positions"
  | "/system/users/tenants"
  | "/system/access/roles"
  | "/system/access/permissions"
  | "/system/access/menus"
  | "/system/security/operation-logs"
  | "/system/security/login-logs"
  | "/system/security/events"
  | "/system/monitoring/sessions"
	| "/system/monitoring/health"
	| "/system/security/block-rules"
  | "/system/notifications/notices"
  | "/system/notifications/messages"
  | "/system/notifications/templates"
  | "/system/notifications/deliveries"
  | "/system/integrations/schedules"
  | "/system/integrations/api-clients"
  | "/system/integrations/webhooks" {
  return (
    value === "/dashboard" ||
    value === "/profile/basic" ||
    value === "/profile/security" ||
    value === "/profile/connections" ||
    value === "/system/settings/configs" ||
    value === "/system/settings/dictionaries" ||
    value === "/system/settings/regions" ||
    value === "/system/settings/modules" ||
    value === "/system/storage/files" ||
    value === "/system/users/accounts" ||
    value === "/system/users/departments" ||
    value === "/system/users/positions" ||
    value === "/system/users/tenants" ||
    value === "/system/access/roles" ||
    value === "/system/access/permissions" ||
    value === "/system/access/menus" ||
    value === "/system/security/operation-logs" ||
    value === "/system/security/login-logs" ||
    value === "/system/security/events" ||
    value === "/system/monitoring/sessions" ||
	value === "/system/monitoring/health" ||
	value === "/system/security/block-rules" ||
    value === "/system/notifications/notices" ||
    value === "/system/notifications/messages" ||
    value === "/system/notifications/templates" ||
    value === "/system/notifications/deliveries" ||
    value === "/system/integrations/schedules" ||
    value === "/system/integrations/api-clients" ||
    value === "/system/integrations/webhooks"
  );
}
