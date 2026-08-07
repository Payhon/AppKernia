import { AppShell } from "../components/AppShell";
import { PermissionBoundary } from "../components/PermissionBoundary";
import { AdminApplicationsPage } from "./AdminApplicationsPage";
import { AdminAppUsersPage } from "./AdminAppUsersPage";
import { AdminAppPagesPage } from "./AdminAppPagesPage";

export function AdminApplicationsRoute() { return <AppShell><PermissionBoundary permission="app.application.read"><AdminApplicationsPage /></PermissionBoundary></AppShell>; }
export function AdminAppUsersRoute() { return <AppShell><PermissionBoundary permission="app.user.read"><AdminAppUsersPage /></PermissionBoundary></AppShell>; }
export function AdminAppPagesRoute() { return <AppShell><PermissionBoundary permission="app.content.read"><AdminAppPagesPage /></PermissionBoundary></AppShell>; }
