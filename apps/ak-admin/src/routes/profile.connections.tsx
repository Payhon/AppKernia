import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { ProfileConnectionsRoute } from '../pages/ProfileConnectionsRoute'

export const Route = createFileRoute('/profile/connections')({ component: () => <ProtectedPage><ProfileConnectionsRoute /></ProtectedPage> })
