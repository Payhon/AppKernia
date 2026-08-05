import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminMobileReleasesPage } from "./AdminMobileReleasesPage";

export function AdminMobileReleasesRoute() {
  return <AppShell><PermissionBoundary permission="mobile.release.read"><AdminMobileReleasesPage /></PermissionBoundary></AppShell>;
}
