import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminWebhooksPage } from "../pages/AdminWebhooksPage";
export const Route=createFileRoute("/system/integrations/webhooks")({component:()=> <ProtectedPage><AppShell><PermissionBoundary permission="sys.webhook.read"><AdminWebhooksPage/></PermissionBoundary></AppShell></ProtectedPage>});
