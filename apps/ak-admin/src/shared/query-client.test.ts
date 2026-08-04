import { describe, expect, it } from 'vitest'

import { purgeTenantScopedQueries, queryClient } from './query-client'

describe('tenant query cache isolation', () => {
  it('removes tenant data and retains explicitly global data', () => {
    queryClient.setQueryData(['tenant', 'old', 'users'], ['secret'])
    queryClient.setQueryData(['admin-users'], ['legacy-tenant-secret'])
    queryClient.setQueryData(['global', 'permission-catalog'], ['safe'])

    purgeTenantScopedQueries(queryClient)

    expect(queryClient.getQueryData(['tenant', 'old', 'users'])).toBeUndefined()
    expect(queryClient.getQueryData(['admin-users'])).toBeUndefined()
    expect(queryClient.getQueryData(['global', 'permission-catalog'])).toEqual(['safe'])
  })
})
