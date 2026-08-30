import { createFileRoute, Navigate } from "@tanstack/react-router";
export const Route = createFileRoute("/system/notifications/deliveries")({ component: () => <Navigate replace to="/system/notifications/operations" /> });
