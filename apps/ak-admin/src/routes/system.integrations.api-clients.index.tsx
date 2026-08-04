import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { FeatureBoundary } from "../components/FeatureBoundary";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminApiClientsPage } from "../pages/AdminApiClientsPage";

export const Route = createFileRoute("/system/integrations/api-clients/")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <ProtectedPage>
      <AppShell>
        <FeatureBoundary feature="api_clients">
          <PermissionBoundary permission="sys.api_client.read">
            <AdminApiClientsPage />
          </PermissionBoundary>
        </FeatureBoundary>
      </AppShell>
    </ProtectedPage>
  );
}
