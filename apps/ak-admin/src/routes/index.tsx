import { createFileRoute } from '@tanstack/react-router'

import { RootRedirect } from '../app/route-boundaries'

export const Route = createFileRoute('/')({ component: RootRedirect })
