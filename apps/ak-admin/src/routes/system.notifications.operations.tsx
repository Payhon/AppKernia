import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminNotificationOperationsRoute } from "../pages/AdminNotificationOperationsRoute";

export const Route = createFileRoute("/system/notifications/operations")({ component: () => <ProtectedPage><AdminNotificationOperationsRoute /></ProtectedPage> });
