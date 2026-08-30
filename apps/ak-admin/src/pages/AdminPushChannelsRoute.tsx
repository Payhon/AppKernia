import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminPushChannelsPage } from "./AdminPushChannelsPage";

export function AdminPushChannelsRoute() {
  return <AppShell><PermissionBoundary permission="notify.push_provider.read"><AdminPushChannelsPage /></PermissionBoundary></AppShell>;
}
