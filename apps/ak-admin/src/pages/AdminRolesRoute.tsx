import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminRolesPage } from './AdminRolesPage'
export function AdminRolesRoute(){return <AppShell><PermissionBoundary permission="iam.role.read"><AdminRolesPage/></PermissionBoundary></AppShell>}
