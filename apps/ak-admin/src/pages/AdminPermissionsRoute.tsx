import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminPermissionsPage } from './AdminPermissionsPage'
export function AdminPermissionsRoute(){return <AppShell><PermissionBoundary permission="iam.permission.read"><AdminPermissionsPage/></PermissionBoundary></AppShell>}
