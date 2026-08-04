import { AppShell } from '../components/AppShell'
import { FeatureBoundary } from '../components/FeatureBoundary'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminTenantsPage } from './AdminTenantsPage'

export function AdminTenantsRoute() { return <AppShell><FeatureBoundary feature="multi_tenant"><PermissionBoundary permission="iam.tenant.read"><AdminTenantsPage /></PermissionBoundary></FeatureBoundary></AppShell> }
