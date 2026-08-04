import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { AdminSecurityEventsRoute } from '../pages/AdminSecurityEventsRoute'

export const Route = createFileRoute('/system/security/events/')({
  component: () => <ProtectedPage><AdminSecurityEventsRoute /></ProtectedPage>,
})
