import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { ProfileConnectionsPage } from './ProfileConnectionsPage'

export function ProfileConnectionsRoute() {
  return <AppShell><PermissionBoundary permission="iam.oauth.manage_self"><ProfileConnectionsPage /></PermissionBoundary></AppShell>
}
