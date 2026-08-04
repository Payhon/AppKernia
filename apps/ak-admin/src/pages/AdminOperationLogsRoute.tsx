import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminOperationLogsPage } from './AdminOperationLogsPage'

export function AdminOperationLogsRoute() { return <AppShell><PermissionBoundary permission="audit.operation.read"><AdminOperationLogsPage/></PermissionBoundary></AppShell> }
