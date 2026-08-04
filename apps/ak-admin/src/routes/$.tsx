import { createFileRoute } from '@tanstack/react-router'

import { ErrorPage } from '../pages/ErrorPage'

export const Route = createFileRoute('/$')({ component: () => <ErrorPage status="404" titleKey="routes.errors.not-found.title" /> })
