import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminShareConfigsRoute } from "../pages/AdminShareConfigsRoute";

export const Route = createFileRoute("/system/settings/share-configs")({
  component: () => (
    <ProtectedPage>
      <AdminShareConfigsRoute />
    </ProtectedPage>
  ),
});
