import { createFileRoute } from '@tanstack/react-router'

import { AnonymousPage } from '../app/route-boundaries'
import { ResetPasswordPage } from '../pages/ResetPasswordPage'

export const Route = createFileRoute('/reset-password')({ component: () => <AnonymousPage featureFlag="password_recovery"><ResetPasswordPage /></AnonymousPage> })
