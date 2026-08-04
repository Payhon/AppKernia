import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminRegionsRoute } from "../pages/AdminRegionsRoute";
export const Route = createFileRoute("/system/settings/regions")({
  component: () => (
    <ProtectedPage>
      <AdminRegionsRoute />
    </ProtectedPage>
  ),
});
