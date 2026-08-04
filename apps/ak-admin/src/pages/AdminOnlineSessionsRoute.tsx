import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminOnlineSessionsPage } from './AdminOnlineSessionsPage'

export function AdminOnlineSessionsRoute() {
  return <AppShell><PermissionBoundary permission="iam.session.read"><AdminOnlineSessionsPage /></PermissionBoundary></AppShell>
}
