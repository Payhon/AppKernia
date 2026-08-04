import { QueryClient, type Query } from '@tanstack/react-query'

export function isTenantScopedQuery(query: Pick<Query, 'queryKey'>): boolean {
  return query.queryKey[0] !== 'global'
}

export function purgeTenantScopedQueries(client: QueryClient): void {
  client.removeQueries({ predicate: isTenantScopedQuery })
}

export const queryClient = new QueryClient({
  defaultOptions: {
    mutations: { retry: false },
    queries: { refetchOnWindowFocus: false, retry: 1, staleTime: 30_000 },
  },
})
