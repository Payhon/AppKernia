import { AppShell } from '../components/AppShell'
import { FeatureBoundary } from '../components/FeatureBoundary'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminTenantDetailPage } from './AdminTenantDetailPage'

export function AdminTenantDetailRoute({ tenantId }: { tenantId: string }) { return <AppShell><FeatureBoundary feature="multi_tenant"><PermissionBoundary permission="iam.tenant.read"><AdminTenantDetailPage tenantId={tenantId} /></PermissionBoundary></FeatureBoundary></AppShell> }
