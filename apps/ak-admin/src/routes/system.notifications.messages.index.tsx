import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminMessagesRoute } from "../pages/AdminNotificationRoutes";
export const Route = createFileRoute("/system/notifications/messages/")({ component: () => <ProtectedPage><AdminMessagesRoute/></ProtectedPage> });
