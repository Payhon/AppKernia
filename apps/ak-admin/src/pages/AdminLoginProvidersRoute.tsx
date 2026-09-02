import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminLoginProvidersPage } from "./AdminLoginProvidersPage";

export function AdminLoginProvidersRoute() {
  return <AppShell><PermissionBoundary permission="sys.login_provider_config.read"><AdminLoginProvidersPage /></PermissionBoundary></AppShell>;
}
