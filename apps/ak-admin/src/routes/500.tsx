import { createFileRoute } from '@tanstack/react-router'

import { ErrorPage } from '../pages/ErrorPage'

export const Route = createFileRoute('/500')({ component: () => <ErrorPage status="500" titleKey="routes.errors.server-error.title" /> })
