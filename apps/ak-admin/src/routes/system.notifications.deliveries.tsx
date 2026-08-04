import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminDeliveriesRoute } from "../pages/AdminNotificationRoutes";
export const Route = createFileRoute("/system/notifications/deliveries")({ component: () => <ProtectedPage><AdminDeliveriesRoute/></ProtectedPage> });
