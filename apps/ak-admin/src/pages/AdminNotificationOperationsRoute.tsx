import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminNotificationOperationsPage } from "./AdminNotificationOperationsPage";

export function AdminNotificationOperationsRoute() {
  return <AppShell><PermissionBoundary permission="notify.observability.read"><AdminNotificationOperationsPage /></PermissionBoundary></AppShell>;
}
