import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminPushChannelsRoute } from "../pages/AdminPushChannelsRoute";

export const Route = createFileRoute("/system/notifications/push-channels")({ component: () => <ProtectedPage><AdminPushChannelsRoute /></ProtectedPage> });
