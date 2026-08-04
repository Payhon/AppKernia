import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminNoticesRoute } from "../pages/AdminNotificationRoutes";
export const Route = createFileRoute("/system/notifications/notices/")({ component: () => <ProtectedPage><AdminNoticesRoute/></ProtectedPage> });
