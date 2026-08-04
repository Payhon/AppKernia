import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminServiceHealthPage } from "../pages/AdminServiceHealthPage";
export const Route=createFileRoute("/system/monitoring/health")({component:()=> <ProtectedPage><AppShell><PermissionBoundary permission="ops.health.read"><AdminServiceHealthPage/></PermissionBoundary></AppShell></ProtectedPage>});
