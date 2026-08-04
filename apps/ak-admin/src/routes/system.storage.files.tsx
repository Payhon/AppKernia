import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/system/storage/files")({ component: Outlet });
