import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminModulesPage } from "./AdminModulesPage";
export function AdminModulesRoute() {
  return (
    <AppShell>
      <PermissionBoundary permission="sys.module.read">
        <AdminModulesPage />
      </PermissionBoundary>
    </AppShell>
  );
}
