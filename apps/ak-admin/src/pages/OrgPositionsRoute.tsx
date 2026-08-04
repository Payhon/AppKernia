import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { OrgPositionsPage } from './OrgPositionsPage'

export function OrgPositionsRoute() { return <AppShell><PermissionBoundary permission="org.position.read"><OrgPositionsPage /></PermissionBoundary></AppShell> }
