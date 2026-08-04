import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminLoginLogsPage } from './AdminLoginLogsPage'

export function AdminLoginLogsRoute() { return <AppShell><PermissionBoundary permission="audit.login.read"><AdminLoginLogsPage/></PermissionBoundary></AppShell> }
