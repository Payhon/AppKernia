import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminUserDetailPage } from './AdminUserDetailPage'

export function AdminUserDetailRoute({userId}:{userId:string}){return <AppShell><PermissionBoundary permission="iam.user.read"><AdminUserDetailPage userId={userId}/></PermissionBoundary></AppShell>}
