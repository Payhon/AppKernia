import { useQuery } from '@tanstack/react-query'

import { authSession } from '../auth/store'

export const selfProfileQueryKey = ['self', 'profile'] as const
export const selfSessionsQueryKey = ['self', 'sessions'] as const
export const selfDevicesQueryKey = ['self', 'devices'] as const

export function selfAvatarQueryKey(path: string | null) {
  return ['self', 'avatar', path] as const
}

export function selfAvatarQueryOptions(path: string) {
  return {
    queryKey: selfAvatarQueryKey(path),
    queryFn: () => authSession.avatarBlob(path),
    staleTime: 5 * 60_000,
  }
}

export function useSelfProfileQuery() {
  return useQuery({
    queryKey: selfProfileQueryKey,
    queryFn: () => authSession.me(),
    staleTime: 30_000,
  })
}

export function useSelfAvatarQuery(path: string | null) {
  return useQuery({
    ...selfAvatarQueryOptions(path ?? ''),
    enabled: path !== null,
  })
}

export function useSelfSessionsQuery() {
  return useQuery({
    queryKey: selfSessionsQueryKey,
    queryFn: () => authSession.selfSessions(),
    staleTime: 15_000,
  })
}

export function useSelfDevicesQuery() {
  return useQuery({
    queryKey: selfDevicesQueryKey,
    queryFn: () => authSession.selfDevices(),
    staleTime: 15_000,
  })
}
