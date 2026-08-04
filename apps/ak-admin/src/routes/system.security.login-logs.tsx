import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminLoginLogsRoute } from '../pages/AdminLoginLogsRoute'

export const Route = createFileRoute('/system/security/login-logs')({ component: () => <ProtectedPage><AdminLoginLogsRoute/></ProtectedPage> })
