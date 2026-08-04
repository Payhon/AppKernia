import { createFileRoute } from "@tanstack/react-router";
import { ProtectedPage } from "../app/route-boundaries";
import { AdminNotificationDetailRoute } from "../pages/AdminNotificationRoutes";
export const Route = createFileRoute("/system/notifications/notices/$noticeId")({ component: Detail });
function Detail(){const{noticeId}=Route.useParams();return <ProtectedPage><AdminNotificationDetailRoute kind="notices" id={noticeId}/></ProtectedPage>}
