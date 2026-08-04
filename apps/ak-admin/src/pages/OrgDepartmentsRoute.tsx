import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { OrgDepartmentsPage } from './OrgDepartmentsPage'

export function OrgDepartmentsRoute() { return <AppShell><PermissionBoundary permission="org.unit.read"><OrgDepartmentsPage /></PermissionBoundary></AppShell> }
