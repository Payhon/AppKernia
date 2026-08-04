import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/system/integrations/schedules")({
  component: Outlet,
});
