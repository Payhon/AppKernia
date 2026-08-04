import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminNotificationMessagesPage } from "./AdminNotificationMessagesPage";
import { AdminNotificationTemplatesPage } from "./AdminNotificationTemplatesPage";
import { AdminNotificationDeliveriesPage } from "./AdminNotificationDeliveriesPage";
import { AdminNotificationDetailPage } from "./AdminNotificationDetailPage";
import type { NotificationKind } from "../features/notifications/hooks";

export function AdminNoticesRoute() { return <AppShell><PermissionBoundary permission="notify.notice.read"><AdminNotificationMessagesPage kind="notices"/></PermissionBoundary></AppShell>; }
export function AdminMessagesRoute() { return <AppShell><PermissionBoundary permission="notify.message.read"><AdminNotificationMessagesPage kind="messages"/></PermissionBoundary></AppShell>; }
export function AdminTemplatesRoute() { return <AppShell><PermissionBoundary permission="notify.template.read"><AdminNotificationTemplatesPage/></PermissionBoundary></AppShell>; }
export function AdminDeliveriesRoute() { return <AppShell><PermissionBoundary permission="notify.delivery.read"><AdminNotificationDeliveriesPage/></PermissionBoundary></AppShell>; }
export function AdminNotificationDetailRoute({ kind, id }: { kind: NotificationKind; id: string }) { const permission=kind==="notices"?"notify.notice.read":"notify.message.read";return <AppShell><PermissionBoundary permission={permission}><AdminNotificationDetailPage kind={kind} id={id}/></PermissionBoundary></AppShell>; }
