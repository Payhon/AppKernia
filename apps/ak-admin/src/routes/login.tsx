import { createFileRoute } from '@tanstack/react-router'

import { AnonymousPage } from '../app/route-boundaries'
import { LoginPage } from '../pages/LoginPage'

export const Route = createFileRoute('/login')({ component: () => <AnonymousPage><LoginPage /></AnonymousPage> })
