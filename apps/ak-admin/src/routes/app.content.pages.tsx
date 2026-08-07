import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminAppPagesRoute } from "../pages/AdminAppRoutes";
export const Route = createFileRoute("/app/content/pages")({ component: () => <ProtectedPage><AdminAppPagesRoute /></ProtectedPage> });
