import { createFileRoute } from '@tanstack/react-router'

import { AnonymousPage } from '../app/route-boundaries'
import { ForgotPasswordPage } from '../pages/ForgotPasswordPage'

export const Route = createFileRoute('/forgot-password')({ component: () => <AnonymousPage featureFlag="password_recovery"><ForgotPasswordPage /></AnonymousPage> })
