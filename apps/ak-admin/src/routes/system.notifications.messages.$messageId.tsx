import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminNotificationDetailRoute } from "../pages/AdminNotificationRoutes";
export const Route = createFileRoute("/system/notifications/messages/$messageId")({ component: Detail });
function Detail(){const{messageId}=Route.useParams();return <ProtectedPage><AdminNotificationDetailRoute kind="messages" id={messageId}/></ProtectedPage>}
