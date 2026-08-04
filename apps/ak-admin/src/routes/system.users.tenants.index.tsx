import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminTenantsRoute } from '../pages/AdminTenantsRoute'

export const Route = createFileRoute('/system/users/tenants/')({ component: () => <ProtectedPage><AdminTenantsRoute /></ProtectedPage> })
