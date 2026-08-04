import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { ProfileSecurityRoute } from '../pages/ProfileSecurityRoute'

export const Route = createFileRoute('/profile/security')({ component: () => <ProtectedPage><ProfileSecurityRoute /></ProtectedPage> })
