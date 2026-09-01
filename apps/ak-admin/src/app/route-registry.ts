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
  "system.settings.share-configs",
  "system.settings.dictionaries",
  "system.settings.regions",
  "system.storage.files",
  "system.notifications.notices",
  "system.notifications.messages",
  "system.notifications.templates",
  "system.notifications.deliveries",
  "system.notifications.operations",
  "system.notifications.push-channels",
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
  "system.content.articles",
  "system.content.categories",
  "system.mobile.releases",
  "app.applications",
  "app.upgrade-center",
  "app.users",
  "app.content.articles",
  "app.content.categories",
  "app.content.pages",
  "app.feedbacks",
]);

export interface ResolvedMenuDirectory {
  code: string;
  type: "directory";
  icon: string | null;
  i18nKey: string;
  path: string | null;
  componentKey: null;
  children: ResolvedMenuItem[];
}

export interface ResolvedMenuPage {
  code: string;
  type: "page";
  icon: string | null;
  i18nKey: string;
  path: string;
  componentKey: string;
  children: [];
}

export type ResolvedMenuItem = ResolvedMenuDirectory | ResolvedMenuPage;

export interface ShellNavigationItems {
  primary: ResolvedMenuItem[];
  system: ResolvedMenuDirectory | null;
}

const navigationRootCodes = new Set(["app", "dashboard", "system"]);

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
  const childrenByParent = new Map<string | null, AdminMenuItem[]>();
  for (const menu of menus) {
    const siblings = childrenByParent.get(menu.parent_id) ?? [];
    siblings.push(menu);
    childrenByParent.set(menu.parent_id, siblings);
  }
  for (const siblings of childrenByParent.values()) {
    siblings.sort((left, right) => left.sort - right.sort || left.code.localeCompare(right.code));
  }

  const resolveNode = (menu: AdminMenuItem, ancestors: ReadonlySet<string>): ResolvedMenuItem | null => {
    if (ancestors.has(menu.id)) return null;
    if (menu.feature_flag && featureFlags[menu.feature_flag] !== true) return null;

    if (menu.type === "page") {
      if (!menu.component_key) return null;
      const route = findRegisteredRoute(menu.component_key);
      if (!route) {
        onUnknown(menu.component_key);
        return null;
      }
      if (!isImplementedRoute(route.componentKey) || !canAccessRoute(route, permissions, featureFlags)) return null;
      return {
        code: menu.code,
        type: "page",
        icon: menu.icon,
        componentKey: route.componentKey,
        i18nKey: menu.i18n_key,
        path: route.path,
        children: [],
      };
    }

    if (menu.type !== "directory") return null;
    const nextAncestors = new Set(ancestors);
    nextAncestors.add(menu.id);
    const children = (childrenByParent.get(menu.id) ?? [])
      .map((child) => resolveNode(child, nextAncestors))
      .filter((child): child is ResolvedMenuItem => child !== null);
    if (children.length === 0) return null;
    return {
      code: menu.code,
      type: "directory",
      icon: menu.icon,
      componentKey: null,
      i18nKey: menu.i18n_key,
      path: menu.path,
      children,
    };
  };

  return (childrenByParent.get(null) ?? [])
    .filter((menu) => navigationRootCodes.has(menu.code))
    .map((menu) => resolveNode(menu, new Set()))
    .filter((menu): menu is ResolvedMenuItem => menu !== null);
}

export function findMenuAncestorKeys(
  menus: readonly ResolvedMenuItem[],
  pathname: string,
  ancestors: readonly string[] = [],
): string[] {
  for (const menu of menus) {
    if (menu.type === "page" && menu.path === pathname) return [...ancestors];
    const match = findMenuAncestorKeys(menu.children, pathname, [...ancestors, menu.code]);
    if (match.length > 0) return match;
  }
  return [];
}

export function partitionShellNavigation(menus: readonly ResolvedMenuItem[]): ShellNavigationItems {
  const system = menus.find((menu): menu is ResolvedMenuDirectory => (
    menu.code === "system" && menu.type === "directory"
  )) ?? null;
  return {
    primary: menus.filter((menu) => menu.code !== "system"),
    system,
  };
}

export function isSystemPath(pathname: string): boolean {
  return pathname === "/system" || pathname.startsWith("/system/");
}

export function flattenMenuPages(menus: readonly ResolvedMenuItem[]): ResolvedMenuPage[] {
  return menus.flatMap((menu) => menu.type === "page" ? [menu] : flattenMenuPages(menu.children));
}

export function isSafeInternalRedirect(
  value: string | null,
): value is
  | "/dashboard"
  | "/app/applications"
  | "/app/users"
  | "/app/content/articles"
  | "/app/content/categories"
  | "/app/content/pages"
  | "/app/feedbacks"
  | "/profile/basic"
  | "/profile/security"
  | "/profile/connections"
  | "/system/settings/configs"
  | "/system/settings/dictionaries"
  | "/system/settings/regions"
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
  | "/system/notifications/operations"
  | "/system/notifications/push-channels"
  | "/system/integrations/schedules"
  | "/system/integrations/api-clients"
  | "/system/integrations/webhooks" {
  return (
    value === "/dashboard" ||
    value === "/app/applications" ||
    value === "/app/users" ||
    value === "/app/content/articles" ||
    value === "/app/content/categories" ||
    value === "/app/content/pages" ||
    value === "/app/feedbacks" ||
    value === "/profile/basic" ||
    value === "/profile/security" ||
    value === "/profile/connections" ||
    value === "/system/settings/configs" ||
    value === "/system/settings/dictionaries" ||
    value === "/system/settings/regions" ||
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
    value === "/system/notifications/operations" ||
    value === "/system/notifications/push-channels" ||
    value === "/system/integrations/schedules" ||
    value === "/system/integrations/api-clients" ||
    value === "/system/integrations/webhooks"
  );
}
