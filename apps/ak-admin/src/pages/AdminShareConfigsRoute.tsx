import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminShareConfigsPage } from "./AdminShareConfigsPage";

export function AdminShareConfigsRoute() {
  return (
    <AppShell>
      <PermissionBoundary permission="sys.share_config.read">
        <AdminShareConfigsPage />
      </PermissionBoundary>
    </AppShell>
  );
}
