import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminJobSchedulesPage } from "./AdminJobSchedulesPage";

export function AdminJobSchedulesRoute() {
  return (
    <AppShell>
      <PermissionBoundary permission="jobs.schedule.read">
        <AdminJobSchedulesPage />
      </PermissionBoundary>
    </AppShell>
  );
}
