import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminRegionsPage } from "./AdminRegionsPage";
export function AdminRegionsRoute() {
  return (
    <AppShell>
      <PermissionBoundary permission="sys.region.read">
        <AdminRegionsPage />
      </PermissionBoundary>
    </AppShell>
  );
}
