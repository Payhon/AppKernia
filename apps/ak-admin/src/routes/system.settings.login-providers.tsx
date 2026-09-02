import { createFileRoute } from "@tanstack/react-router";

import { ProtectedPage } from "../app/route-boundaries";
import { AdminLoginProvidersRoute } from "../pages/AdminLoginProvidersRoute";

export const Route = createFileRoute("/system/settings/login-providers")({
  component: () => <ProtectedPage><AdminLoginProvidersRoute /></ProtectedPage>,
});
