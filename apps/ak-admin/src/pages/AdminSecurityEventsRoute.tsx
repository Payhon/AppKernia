import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminSecurityEventsPage } from './AdminSecurityEventsPage'

export function AdminSecurityEventsRoute() { return <AppShell><PermissionBoundary permission="audit.security.read"><AdminSecurityEventsPage/></PermissionBoundary></AppShell> }
