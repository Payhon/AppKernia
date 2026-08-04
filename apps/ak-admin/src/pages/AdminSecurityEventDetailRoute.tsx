import { AppShell } from '../components/AppShell'
import { PermissionBoundary } from '../components/PermissionBoundary'
import { AdminSecurityEventDetailPage } from './AdminSecurityEventDetailPage'

export function AdminSecurityEventDetailRoute({ eventId }: { eventId: string }) { return <AppShell><PermissionBoundary permission="audit.security.read"><AdminSecurityEventDetailPage eventId={eventId}/></PermissionBoundary></AppShell> }
