import { createFileRoute } from '@tanstack/react-router'

import { AnonymousPage } from '../app/route-boundaries'
import { RegisterPage } from '../pages/RegisterPage'

export const Route = createFileRoute('/register')({ component: () => <AnonymousPage featureFlag="admin_registration"><RegisterPage /></AnonymousPage> })
