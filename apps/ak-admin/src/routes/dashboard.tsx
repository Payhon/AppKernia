import { createFileRoute } from '@tanstack/react-router'

import { ProtectedPage } from '../app/route-boundaries'
import { DashboardRoute } from '../pages/DashboardRoute'

export const Route = createFileRoute('/dashboard')({
  validateSearch: (search: Record<string, unknown>): { range: '7d' | '30d' | '90d' } => ({
    range: search['range'] === '7d' || search['range'] === '90d' ? search['range'] : '30d',
  }),
  component: () => <ProtectedPage><DashboardRoute /></ProtectedPage>,
})
