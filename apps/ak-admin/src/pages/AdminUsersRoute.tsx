import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminUsersPage } from './AdminUsersPage'

export function AdminUsersRoute(){return <AppShell><PermissionBoundary permission="iam.user.read"><AdminUsersPage/></PermissionBoundary></AppShell>}
