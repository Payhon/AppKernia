import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminTemplatesRoute } from "../pages/AdminNotificationRoutes";
export const Route = createFileRoute("/system/notifications/templates")({ component: () => <ProtectedPage><AdminTemplatesRoute/></ProtectedPage> });
