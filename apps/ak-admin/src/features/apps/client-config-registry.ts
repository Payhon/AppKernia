export const clientConfigTabPermissions = [
  { id: "share", readPermission: "app.share_binding.read", updatePermission: "app.share_binding.update" },
  { id: "scanner", readPermission: "app.scanner_config.read", updatePermission: "app.scanner_config.update" },
  { id: "login-providers", readPermission: "app.login_provider_binding.read", updatePermission: "app.login_provider_binding.update" },
] as const;

export type ClientConfigTabId = (typeof clientConfigTabPermissions)[number]["id"];

export function canReadAnyClientConfig(permissions: ReadonlySet<string>): boolean {
  return clientConfigTabPermissions.some((tab) => permissions.has(tab.readPermission));
}
