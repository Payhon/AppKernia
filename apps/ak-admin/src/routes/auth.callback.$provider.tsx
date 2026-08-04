import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { OAuthCallbackPage } from '../pages/OAuthCallbackPage'

export const Route = createFileRoute('/auth/callback/$provider')({ component: () => <ProtectedPage><OAuthCallbackPage /></ProtectedPage> })
