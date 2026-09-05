const markerName = 'ak-admin-base-path'

export function normalizeAdminBasePath(value: string | null | undefined): string {
  const path = value?.trim() ?? '/'
  if (!path.startsWith('/') || path.includes('?') || path.includes('#')) return '/'
  const normalized = path.replace(/\/{2,}/g, '/').replace(/\/$/, '')
  return normalized || '/'
}

export const adminBasePath = normalizeAdminBasePath(
  typeof document === 'undefined'
    ? undefined
    : document.querySelector<HTMLMetaElement>(`meta[name="${markerName}"]`)?.content,
)

export function withAdminBasePath(path: string, basePath = adminBasePath): string {
  if (!path.startsWith('/')) throw new Error('Admin paths must start with /')
  const base = normalizeAdminBasePath(basePath)
  if (base === '/') return path
  return path === '/' ? `${base}/` : `${base}${path}`
}

export function withoutAdminBasePath(path: string, basePath = adminBasePath): string {
  const base = normalizeAdminBasePath(basePath)
  if (base === '/') return path
  if (path === base || path === `${base}/`) return '/'
  return path.startsWith(`${base}/`) ? path.slice(base.length) : path
}
