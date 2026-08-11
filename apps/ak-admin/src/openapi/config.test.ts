import { describe, expect, it, vi } from 'vitest'

import {
  dismissOpenApiNotice,
  isOpenApiNoticeDismissed,
  normalizeOpenApiLocale,
  openApiDocsHref,
  openApiNoticeDismissedStorageKey,
  resolveOpenApiLocale,
  scalarLocale,
} from './config'
import { createOpenApiFetch, localizeOpenApiDocument } from './reference'

const referenceTranslations = {
  'api_reference.surface.platform_public': '平台与公共接口',
  'api_reference.module.platform_health': '健康检查',
  'api_reference.operation.enableAdminUser': '启用租户用户',
  'api_reference.operation.getLiveness': '检查 API 进程存活状态',
}

const canonicalFixture = `openapi: 3.1.0
tags:
  - name: platform-health
    x-displayName: Platform health
    x-appkernia-i18n-key: api_reference.module.platform_health
x-tagGroups:
  - name: Platform and Public APIs
    x-appkernia-i18n-key: api_reference.surface.platform_public
    tags: [platform-health]
paths:
  /internal/v1/health/live:
    get:
      operationId: getLiveness
      summary: Check API process liveness
      tags: [platform-health]
      responses: {}
components:
  pathItems:
    AdminUserEnable:
      post:
        operationId: enableAdminUser
        summary: Enable a suspended tenant membership
        tags: [platform-health]
        responses: {}
`

describe('OpenAPI documentation configuration', () => {
  it('normalizes only the supported application locales', () => {
    expect(normalizeOpenApiLocale('en-GB')).toBe('en-US')
    expect(normalizeOpenApiLocale('zh-Hans')).toBe('zh-CN')
    expect(normalizeOpenApiLocale('fr-FR')).toBe('zh-CN')
    expect(scalarLocale('en-US')).toBe('en')
    expect(scalarLocale('zh-CN')).toBe('zh-CN')
  })

  it('prefers the explicit query locale and generates a same-origin docs link', () => {
    expect(resolveOpenApiLocale('?lang=en-US', 'zh-CN')).toBe('en-US')
    expect(resolveOpenApiLocale('', 'en-AU')).toBe('en-US')
    expect(openApiDocsHref('zh-CN')).toBe('/openapi/?lang=zh-CN')
  })

  it('dismisses the interactive testing notice only for the current browsing session', () => {
    const values = new Map<string, string>()
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => {
        values.set(key, value)
      },
    }

    expect(isOpenApiNoticeDismissed(storage)).toBe(false)
    dismissOpenApiNotice(storage)
    expect(values.get(openApiNoticeDismissedStorageKey)).toBe('1')
    expect(isOpenApiNoticeDismissed(storage)).toBe(true)
  })

  it('omits browser credentials and sends the canonical Accept-Language', async () => {
    const response = new Response(null, { status: 204 })
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(response)
    const docsFetch = createOpenApiFetch('en-US', referenceTranslations, fetchImplementation)

    await docsFetch('/admin-api/v1/auth/context', {
      credentials: 'include',
      headers: { Authorization: 'Bearer test-only-token' },
    })

    const [, init] = fetchImplementation.mock.calls[0] ?? []
    expect(init?.credentials).toBe('omit')
    expect(new Headers(init?.headers).get('Accept-Language')).toBe('en-US')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer test-only-token')
  })

  it('localizes surface, module, sidebar, body, and search title inputs without changing stable identifiers', () => {
    const document = localizeOpenApiDocument(canonicalFixture, referenceTranslations)
    const tags = document['tags'] as Record<string, unknown>[]
    const groups = document['x-tagGroups'] as Record<string, unknown>[]
    const operation = ((document['paths'] as Record<string, Record<string, unknown>>)['/internal/v1/health/live']?.['get']) as Record<string, unknown>
    const componentOperation = (((document['components'] as Record<string, unknown>)['pathItems'] as Record<string, Record<string, unknown>>)['AdminUserEnable']?.['post']) as Record<string, unknown>

    expect(groups[0]?.['name']).toBe('平台与公共接口')
    expect(tags[0]?.['x-displayName']).toBe('健康检查')
    expect(operation['summary']).toBe('检查 API 进程存活状态')
    expect(operation['operationId']).toBe('getLiveness')
    expect(operation['tags']).toEqual(['platform-health'])
    expect(componentOperation['summary']).toBe('启用租户用户')
  })

  it('fails closed when a tag or title translation is missing', () => {
    expect(() => localizeOpenApiDocument(canonicalFixture, {})).toThrow('OPENAPI_TRANSLATION_MISSING')
    expect(() => localizeOpenApiDocument(canonicalFixture, {
      ...referenceTranslations,
      'api_reference.operation.getLiveness': '',
    })).toThrow('OPENAPI_TRANSLATION_MISSING:api_reference.operation.getLiveness')
  })

  it('fails closed for duplicate operation IDs and operations without exactly one registered module', () => {
    const duplicate = canonicalFixture.replace('paths:', `paths:\n  /duplicate:\n    get:\n      operationId: getLiveness\n      summary: Duplicate\n      tags: [platform-health]\n      responses: {}`)
    expect(() => localizeOpenApiDocument(duplicate, referenceTranslations)).toThrow('OPENAPI_OPERATION_ID_INVALID')
    expect(() => localizeOpenApiDocument(canonicalFixture.replace('tags: [platform-health]\n      responses', 'tags: []\n      responses'), referenceTranslations)).toThrow('OPENAPI_OPERATION_TAG_INVALID:getLiveness')
  })

  it('returns a localized in-memory document only for the canonical Scalar fetch', async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(new Response(canonicalFixture, {
      headers: { 'Content-Type': 'application/yaml', ETag: 'canonical-etag' },
      status: 200,
    }))
    const docsFetch = createOpenApiFetch('zh-CN', referenceTranslations, fetchImplementation)

    const response = await docsFetch('/openapi/openapi.yaml')
    const document = await response.json() as { paths: Record<string, { get: { summary: string } }> }

    expect(response.headers.get('Content-Type')).toBe('application/json; charset=utf-8')
    expect(response.headers.get('ETag')).toBeNull()
    expect(document.paths['/internal/v1/health/live']?.get.summary).toBe('检查 API 进程存活状态')
    const [, init] = fetchImplementation.mock.calls[0] ?? []
    expect(init?.credentials).toBe('omit')
    expect(new Headers(init?.headers).get('Accept-Language')).toBe('zh-CN')
  })
})
