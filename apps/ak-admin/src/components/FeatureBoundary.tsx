import type { PropsWithChildren } from 'react'

import { useAuthStore } from '../features/auth/store'
import { ErrorPage } from '../pages/ErrorPage'

export function FeatureBoundary({ feature, children }: PropsWithChildren<{ feature: string }>) {
  const enabled = useAuthStore((state) => state.context?.feature_flags[feature] === true)
  if (!enabled) return <ErrorPage status="404" titleKey="routes.errors.not-found.title" />
  return <>{children}</>
}
