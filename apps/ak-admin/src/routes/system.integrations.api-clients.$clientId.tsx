import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { FeatureBoundary } from "../components/FeatureBoundary";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminApiClientDetailPage } from "../pages/AdminApiClientDetailPage";

export const Route = createFileRoute(
  "/system/integrations/api-clients/$clientId",
)({ component: RouteComponent });

function RouteComponent() {
  const { clientId } = Route.useParams();
  return (
    <ProtectedPage>
      <AppShell>
        <FeatureBoundary feature="api_clients">
          <PermissionBoundary permission="sys.api_client.read">
            <AdminApiClientDetailPage clientId={clientId} />
          </PermissionBoundary>
        </FeatureBoundary>
      </AppShell>
    </ProtectedPage>
  );
}
