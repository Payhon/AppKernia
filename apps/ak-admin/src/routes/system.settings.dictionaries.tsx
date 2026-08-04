import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminDictionariesRoute } from "../pages/AdminDictionariesRoute";

export const Route = createFileRoute("/system/settings/dictionaries")({
  component: () => (
    <ProtectedPage>
      <AdminDictionariesRoute />
    </ProtectedPage>
  ),
});
