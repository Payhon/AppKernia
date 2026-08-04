import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminConfigsPage } from "./AdminConfigsPage";

export function AdminConfigsRoute() {
  return (
    <AppShell>
      <PermissionBoundary permission="sys.config.read">
        <AdminConfigsPage />
      </PermissionBoundary>
    </AppShell>
  );
}
