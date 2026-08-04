import { createFileRoute } from '@tanstack/react-router'

import { ErrorPage } from '../pages/ErrorPage'

export const Route = createFileRoute('/403')({ component: () => <ErrorPage status="403" titleKey="routes.errors.forbidden.title" /> })
