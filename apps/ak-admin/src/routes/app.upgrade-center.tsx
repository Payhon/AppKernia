import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminMobileReleasesRoute } from "../pages/AdminMobileReleasesRoute";

export const Route = createFileRoute("/app/upgrade-center")({
  component: () => <ProtectedPage><AdminMobileReleasesRoute /></ProtectedPage>,
});
