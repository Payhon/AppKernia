import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { ProfileBasicRoute } from '../pages/ProfileBasicRoute'

export const Route = createFileRoute('/profile/basic')({ component: () => <ProtectedPage><ProfileBasicRoute /></ProtectedPage> })
