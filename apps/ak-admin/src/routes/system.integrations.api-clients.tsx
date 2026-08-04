import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/system/integrations/api-clients")({
  component: Outlet,
});
