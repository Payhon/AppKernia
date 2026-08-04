import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminModulesRoute } from "../pages/AdminModulesRoute";
export const Route = createFileRoute("/system/settings/modules")({
  component: () => (
    <ProtectedPage>
      <AdminModulesRoute />
    </ProtectedPage>
  ),
});
