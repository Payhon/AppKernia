import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminApplicationsRoute } from "../pages/AdminAppRoutes";
export const Route = createFileRoute("/app/applications")({ component: () => <ProtectedPage><AdminApplicationsRoute /></ProtectedPage> });
