import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { AdminOnlineSessionsRoute } from '../pages/AdminOnlineSessionsRoute'

export const Route = createFileRoute('/system/monitoring/sessions')({
  component: () => <ProtectedPage><AdminOnlineSessionsRoute /></ProtectedPage>,
})
