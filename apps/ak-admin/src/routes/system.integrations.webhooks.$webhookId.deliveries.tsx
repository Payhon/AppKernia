import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminWebhookDeliveriesPage } from "../pages/AdminWebhooksPage";
export const Route=createFileRoute("/system/integrations/webhooks/$webhookId/deliveries")({component:RouteComponent});
function RouteComponent(){const{webhookId}=Route.useParams();return <ProtectedPage><AppShell><PermissionBoundary permission="sys.webhook.delivery.read"><AdminWebhookDeliveriesPage webhookId={webhookId}/></PermissionBoundary></AppShell></ProtectedPage>}
