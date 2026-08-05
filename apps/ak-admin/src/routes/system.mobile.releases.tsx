import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminMobileReleasesRoute } from "../pages/AdminMobileReleasesRoute";

export const Route = createFileRoute("/system/mobile/releases")({
  component: () => <ProtectedPage><AdminMobileReleasesRoute /></ProtectedPage>,
});
