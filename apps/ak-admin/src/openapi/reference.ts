import { parse } from 'yaml'

import type { OpenApiLocale } from './config'

type OpenApiObject = Record<string, unknown>

const operationMethods = new Set(['get', 'post', 'put', 'patch', 'delete', 'options', 'head', 'trace'])

function objectValue(value: unknown, errorCode: string): OpenApiObject {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(errorCode)
  return value as OpenApiObject
}

function translatedValue(translations: Readonly<Record<string, string>>, key: unknown): string {
  if (typeof key !== 'string' || key.length === 0) throw new Error('OPENAPI_I18N_KEY_MISSING')
  const value = translations[key]
  if (typeof value !== 'string' || value.trim().length === 0) throw new Error(`OPENAPI_TRANSLATION_MISSING:${key}`)
  return value
}

export function localizeOpenApiDocument(source: string, translations: Readonly<Record<string, string>>): OpenApiObject {
  const document = objectValue(parse(source), 'OPENAPI_DOCUMENT_INVALID')
  if (typeof document['openapi'] !== 'string' || !document['openapi'].startsWith('3.1.')) {
    throw new Error('OPENAPI_DOCUMENT_VERSION_UNSUPPORTED')
  }

  const declaredTags = new Set<string>()
  if (!Array.isArray(document['tags']) || document['tags'].length === 0) throw new Error('OPENAPI_TAGS_MISSING')
  for (const rawTag of document['tags']) {
    const tag = objectValue(rawTag, 'OPENAPI_TAG_INVALID')
    if (typeof tag['name'] !== 'string' || tag['name'].length === 0 || declaredTags.has(tag['name'])) {
      throw new Error('OPENAPI_TAG_NAME_INVALID')
    }
    declaredTags.add(tag['name'])
    tag['x-displayName'] = translatedValue(translations, tag['x-appkernia-i18n-key'])
  }

  if (!Array.isArray(document['x-tagGroups']) || document['x-tagGroups'].length === 0) {
    throw new Error('OPENAPI_TAG_GROUPS_MISSING')
  }
  for (const rawGroup of document['x-tagGroups']) {
    const group = objectValue(rawGroup, 'OPENAPI_TAG_GROUP_INVALID')
    group['name'] = translatedValue(translations, group['x-appkernia-i18n-key'])
  }

  const operationIds = new Set<string>()
  const localizeOperation = (rawOperation: unknown): void => {
    const operation = objectValue(rawOperation, 'OPENAPI_OPERATION_INVALID')
    if (typeof operation['operationId'] !== 'string' || operation['operationId'].length === 0 || operationIds.has(operation['operationId'])) {
      throw new Error('OPENAPI_OPERATION_ID_INVALID')
    }
    operationIds.add(operation['operationId'])
    if (!Array.isArray(operation['tags']) || operation['tags'].length !== 1 || !declaredTags.has(operation['tags'][0] as string)) {
      throw new Error(`OPENAPI_OPERATION_TAG_INVALID:${operation['operationId']}`)
    }
    operation['summary'] = translatedValue(translations, `api_reference.operation.${operation['operationId']}`)
  }
  const localizePathItem = (rawPathItem: unknown): void => {
    const pathItem = objectValue(rawPathItem, 'OPENAPI_PATH_ITEM_INVALID')
    for (const [method, rawOperation] of Object.entries(pathItem)) {
      if (operationMethods.has(method)) localizeOperation(rawOperation)
    }
  }

  const paths = objectValue(document['paths'], 'OPENAPI_PATHS_MISSING')
  for (const rawPathItem of Object.values(paths)) localizePathItem(rawPathItem)

  if (document['components'] !== undefined) {
    const components = objectValue(document['components'], 'OPENAPI_COMPONENTS_INVALID')
    if (components['pathItems'] !== undefined) {
      const pathItems = objectValue(components['pathItems'], 'OPENAPI_COMPONENT_PATH_ITEMS_INVALID')
      for (const rawPathItem of Object.values(pathItems)) localizePathItem(rawPathItem)
    }
  }
  if (operationIds.size === 0) throw new Error('OPENAPI_OPERATIONS_MISSING')
  return document
}

function isCanonicalSpecRequest(input: RequestInfo | URL, init: RequestInit | undefined): boolean {
  const method = (init?.method ?? (input instanceof Request ? input.method : 'GET')).toUpperCase()
  if (method !== 'GET') return false
  const url = input instanceof Request ? input.url : input.toString()
  return new URL(url, 'http://appkernia.local').pathname === '/openapi/openapi.yaml'
}

function localizedResponse(response: Response, document: OpenApiObject): Response {
  const headers = new Headers(response.headers)
  for (const header of ['content-encoding', 'content-length', 'content-type', 'etag', 'last-modified']) headers.delete(header)
  headers.set('Cache-Control', 'no-cache, must-revalidate')
  headers.set('Content-Type', 'application/json; charset=utf-8')
  return new Response(JSON.stringify(document), {
    headers,
    status: response.status,
    statusText: response.statusText,
  })
}

export function createOpenApiFetch(
  locale: OpenApiLocale,
  translations: Readonly<Record<string, string>>,
  fetchImplementation: typeof fetch = globalThis.fetch,
): typeof fetch {
  return async (input, init) => {
    const headers = new Headers(input instanceof Request ? input.headers : undefined)
    new Headers(init?.headers).forEach((value, key) => { headers.set(key, value) })
    headers.set('Accept-Language', locale)

    const response = await fetchImplementation(input, {
      ...init,
      credentials: 'omit',
      headers,
    })
    if (!response.ok || !isCanonicalSpecRequest(input, init)) return response
    const source = await response.text()
    return localizedResponse(response, localizeOpenApiDocument(source, translations))
  }
}
