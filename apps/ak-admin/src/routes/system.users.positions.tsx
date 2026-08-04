import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { OrgPositionsRoute } from '../pages/OrgPositionsRoute'

export const Route = createFileRoute('/system/users/positions')({ component: () => <ProtectedPage><OrgPositionsRoute /></ProtectedPage> })
