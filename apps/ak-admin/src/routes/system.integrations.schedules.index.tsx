import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AdminJobSchedulesRoute } from "../pages/AdminJobScheduleRoutes";

export const Route = createFileRoute("/system/integrations/schedules/")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <ProtectedPage>
      <AdminJobSchedulesRoute />
    </ProtectedPage>
  );
}
