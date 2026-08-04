import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminSecurityEventDetailRoute } from '../pages/AdminSecurityEventDetailRoute'

export const Route = createFileRoute('/system/security/events/$eventId')({ component: EventDetail })
function EventDetail() { const { eventId } = Route.useParams(); return <ProtectedPage><AdminSecurityEventDetailRoute eventId={eventId}/></ProtectedPage> }
