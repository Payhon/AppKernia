import { parseAdminEnvironment } from '../config/env'

export interface RequestContext {
  accessToken?: string
  locale: 'zh-CN' | 'en-US'
}

function requestId(): string {
  return globalThis.crypto.randomUUID()
}

export function createAdminApiClient(readContext: () => RequestContext) {
  const environment = parseAdminEnvironment(import.meta.env)
  return new AdminHttpClient(environment.VITE_AK_API_BASE_URL, readContext)
}

export type AdminApiClient = ReturnType<typeof createAdminApiClient>

class AdminHttpClient {
  readonly #baseUrl: string
  readonly #readContext: () => RequestContext

  constructor(baseUrl: string, readContext: () => RequestContext) {
    this.#baseUrl = baseUrl.replace(/\/$/, '')
    this.#readContext = readContext
  }

  request(path: `/${string}`, init: RequestInit = {}): Promise<Response> {
    const context = this.#readContext()
    const headers = new Headers(init.headers)
    headers.set('Accept-Language', context.locale)
    headers.set('X-Request-ID', requestId())
    if (context.accessToken) {
      headers.set('Authorization', `Bearer ${context.accessToken}`)
    }
    return fetch(`${this.#baseUrl}${path}`, { ...init, credentials: 'include', headers })
  }
}
