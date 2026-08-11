export const openApiLocales = ['zh-CN', 'en-US'] as const
export const openApiNoticeDismissedStorageKey = 'appkernia.openapi.notice.dismissed'

export type OpenApiLocale = (typeof openApiLocales)[number]

export function normalizeOpenApiLocale(value: string | null | undefined): OpenApiLocale {
  return value?.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN'
}

export function resolveOpenApiLocale(search: string, browserLocale: string | null | undefined): OpenApiLocale {
  const requested = new URLSearchParams(search).get('lang')
  return normalizeOpenApiLocale(requested ?? browserLocale)
}

export function scalarLocale(locale: OpenApiLocale): 'en' | 'zh-CN' {
  return locale === 'en-US' ? 'en' : 'zh-CN'
}

export function openApiDocsHref(locale: OpenApiLocale): string {
  return `/openapi/?lang=${encodeURIComponent(locale)}`
}

export function isOpenApiNoticeDismissed(storage: Pick<Storage, 'getItem'> | null | undefined): boolean {
  try {
    return storage?.getItem(openApiNoticeDismissedStorageKey) === '1'
  } catch {
    return false
  }
}

export function dismissOpenApiNotice(storage: Pick<Storage, 'setItem'> | null | undefined): void {
  try {
    storage?.setItem(openApiNoticeDismissedStorageKey, '1')
  } catch {
    // Session storage can be unavailable in privacy-restricted browsing contexts.
  }
}
