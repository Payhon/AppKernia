import { useQuery } from '@tanstack/react-query'

import type { DashboardRange } from '../auth/session'
import { authSession } from '../auth/store'

export function useDashboardSummary(range: DashboardRange) {
  return useQuery({
    queryKey: ['dashboard', 'summary', range],
    queryFn: () => authSession.dashboardSummary(range),
    staleTime: 30_000,
  })
}

export function useDashboardTrends(range: DashboardRange) {
  return useQuery({
    queryKey: ['dashboard', 'trends', range],
    queryFn: () => authSession.dashboardTrends(range),
    staleTime: 30_000,
  })
}

export function useDashboardActivity(range: DashboardRange) {
  return useQuery({
    queryKey: ['dashboard', 'activity', range],
    queryFn: () => authSession.dashboardActivity(range),
    staleTime: 30_000,
  })
}
