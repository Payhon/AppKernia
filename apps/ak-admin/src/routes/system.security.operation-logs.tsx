import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminOperationLogsRoute } from '../pages/AdminOperationLogsRoute'

export const Route = createFileRoute('/system/security/operation-logs')({ component: () => <ProtectedPage><AdminOperationLogsRoute/></ProtectedPage> })
