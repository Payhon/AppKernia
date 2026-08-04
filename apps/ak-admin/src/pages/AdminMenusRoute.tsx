import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminMenusPage } from './AdminMenusPage'
export function AdminMenusRoute(){return <AppShell><PermissionBoundary permission="sys.menu.read"><AdminMenusPage/></PermissionBoundary></AppShell>}
