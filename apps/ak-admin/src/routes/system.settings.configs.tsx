import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminConfigsRoute } from "../pages/AdminConfigsRoute";

export const Route = createFileRoute("/system/settings/configs")({
  component: () => (
    <ProtectedPage>
      <AdminConfigsRoute />
    </ProtectedPage>
  ),
});
