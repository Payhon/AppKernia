export interface AdminRouteRegistration {
  readonly to: string;
  readonly menuCode?: string;
  readonly requiredPermissions: readonly string[];
  readonly featureFlag?: string;
}

export const adminRouteRegistry = {
  dashboard: { to: '/dashboard', menuCode: 'dashboard', requiredPermissions: [] },
  'system.users.accounts': { to: '/system/users/accounts', menuCode: 'system.users.accounts', requiredPermissions: ['iam.user.read'] },
  'system.access.roles': { to: '/system/access/roles', menuCode: 'system.access.roles', requiredPermissions: ['iam.role.read'] },
} as const satisfies Record<string, AdminRouteRegistration>;

export type AdminComponentKey = keyof typeof adminRouteRegistry;
