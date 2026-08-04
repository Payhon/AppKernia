import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminTenantDetailRoute } from '../pages/AdminTenantDetailRoute'

export const Route = createFileRoute('/system/users/tenants/$tenantId')({ component: TenantDetail })
function TenantDetail() { const { tenantId } = Route.useParams(); return <ProtectedPage><AdminTenantDetailRoute tenantId={tenantId} /></ProtectedPage> }
