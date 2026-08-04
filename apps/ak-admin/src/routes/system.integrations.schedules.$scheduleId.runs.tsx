import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminJobScheduleRunsPage } from "../pages/AdminJobScheduleRunsPage";

export const Route = createFileRoute(
  "/system/integrations/schedules/$scheduleId/runs",
)({ component: RouteComponent });

function RouteComponent() {
  const { scheduleId } = Route.useParams();
  return (
    <ProtectedPage>
      <AppShell>
        <PermissionBoundary permission="jobs.run.read">
          <AdminJobScheduleRunsPage scheduleId={scheduleId} />
        </PermissionBoundary>
      </AppShell>
    </ProtectedPage>
  );
}
