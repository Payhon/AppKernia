import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import type { AdminOrgPositionRequest, AdminOrgUnitMoveRequest, AdminOrgUnitRequest } from '../../generated/api/types.gen'
import { authSession } from '../auth/store'

export const orgUnitQueryKey = ['org', 'units'] as const
export const orgPositionQueryKey = ['org', 'positions'] as const

export function useOrgUnits(query: string, status: string) {
  return useQuery({ queryKey: [...orgUnitQueryKey, query, status], queryFn: () => authSession.orgUnitTree(query, status), staleTime: 15_000 })
}

export function useOrgPositions(query: string, status: string, unitId: string) {
  return useQuery({ queryKey: [...orgPositionQueryKey, query, status, unitId], queryFn: () => authSession.orgPositions(query, status, unitId), staleTime: 15_000 })
}

export function useOrgUnitMutations() {
  const client = useQueryClient()
  const invalidate = () => client.invalidateQueries({ queryKey: orgUnitQueryKey })
  return {
    create: useMutation({ mutationFn: (input: AdminOrgUnitRequest) => authSession.createOrgUnit(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminOrgUnitRequest }) => authSession.updateOrgUnit(id, input), onSuccess: invalidate }),
    move: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminOrgUnitMoveRequest }) => authSession.moveOrgUnit(id, input), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: (id: string) => authSession.deleteOrgUnit(id), onSuccess: invalidate }),
  }
}

export function useOrgPositionMutations() {
  const client = useQueryClient()
  const invalidate = () => client.invalidateQueries({ queryKey: orgPositionQueryKey })
  return {
    create: useMutation({ mutationFn: (input: AdminOrgPositionRequest) => authSession.createOrgPosition(input), onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, input }: { id: string; input: AdminOrgPositionRequest }) => authSession.updateOrgPosition(id, input), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: (id: string) => authSession.deleteOrgPosition(id), onSuccess: invalidate }),
  }
}
