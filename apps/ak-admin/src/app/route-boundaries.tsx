import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { lazy, Suspense, useEffect, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { authSession, useAuthStore } from '../features/auth/store'

const ErrorPage = lazy(async () => import('../pages/ErrorPage').then((module) => ({ default: module.ErrorPage })))

export function RootRedirect() {
  const navigate = useNavigate()
  const status = useAuthStore((state) => state.status)
  useEffect(() => {
    if (status === 'authenticated') void navigate({ to: '/dashboard', search: { range: '30d' }, replace: true })
    if (status === 'anonymous') void navigate({ to: '/login', replace: true })
  }, [navigate, status])
  return null
}

export function AuthBootstrap({ children }: { children: ReactNode }) {
  const initialize = useAuthStore((state) => state.initialize)
  useEffect(() => { void initialize() }, [initialize])
  return <>{children}</>
}

export function PageFallback() {
  const { t } = useTranslation()
  return <main className="ak-centered-state" aria-live="polite"><span className="ak-loading-indicator" /><span>{t('common.states.loading')}</span></main>
}

export function ProtectedPage({ children }: { children: ReactNode }) {
  const navigate = useNavigate()
  const status = useAuthStore((state) => state.status)
  useEffect(() => {
    if (status === 'anonymous') {
      const redirect = window.location.pathname
      void navigate({ to: '/login', search: { redirect }, replace: true })
    }
  }, [navigate, status])
  if (status !== 'authenticated') return <PageFallback />
  return <Suspense fallback={<PageFallback />}>{children}</Suspense>
}

export function LazyPage({ children }: { children: ReactNode }) {
  return <Suspense fallback={<PageFallback />}>{children}</Suspense>
}

export function AnonymousPage({ children, featureFlag }: { children: ReactNode; featureFlag?: 'admin_registration' | 'password_recovery' }) {
  const navigate = useNavigate()
  const status = useAuthStore((state) => state.status)
  const publicConfig = useQuery({
    queryKey: ['auth', 'public-config', 'feature-gates'],
    queryFn: () => authSession.publicConfig(),
    enabled: featureFlag !== undefined,
    retry: 1,
    staleTime: 60_000,
  })
  useEffect(() => {
    if (status === 'authenticated') void navigate({ to: '/dashboard', search: { range: '30d' }, replace: true })
  }, [navigate, status])
  if (status === 'bootstrapping' || status === 'authenticated' || (featureFlag !== undefined && publicConfig.isPending)) return <PageFallback />
  if (featureFlag !== undefined && publicConfig.isError) return <LazyPage><ErrorPage status="500" titleKey="routes.errors.server-error.title" /></LazyPage>
  if (featureFlag !== undefined && publicConfig.data?.feature_flags[featureFlag] !== true) return <LazyPage><ErrorPage status="404" titleKey="routes.errors.not-found.title" /></LazyPage>
  return <>{children}</>
}
