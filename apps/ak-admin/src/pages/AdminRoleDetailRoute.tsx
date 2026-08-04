import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminRoleDetailPage } from './AdminRoleDetailPage'
export function AdminRoleDetailRoute({roleId}:{roleId:string}){return <AppShell><PermissionBoundary permission="iam.role.read"><AdminRoleDetailPage roleId={roleId}/></PermissionBoundary></AppShell>}
