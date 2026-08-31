import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { parse } from 'yaml'

const root = resolve(import.meta.dirname, '../../..')
const canonicalPath = resolve(root, 'server/openapi/openapi.yaml')
const operationMethods = new Set(['get', 'post', 'put', 'patch', 'delete', 'options', 'head', 'trace'])

const surfaces = [
  {
    code: 'platform_public',
    name: 'Platform and Public APIs',
    tags: ['platform-health', 'public-app', 'public-content', 'public-dictionary', 'api-client-auth', 'api-client-notifications'],
  },
  {
    code: 'mobile',
    name: 'Mobile APIs',
    tags: ['mobile-auth', 'mobile-profile', 'mobile-devices-sessions', 'mobile-notifications', 'mobile-security', 'mobile-content'],
  },
  {
    code: 'admin',
    name: 'Admin APIs',
    tags: ['admin-auth-profile', 'admin-dashboard', 'admin-app-management', 'admin-releases', 'admin-app-content', 'admin-app-communications', 'admin-app-users', 'admin-organization', 'admin-tenants', 'admin-users', 'admin-access-control', 'admin-system-settings', 'admin-share-configuration', 'admin-storage', 'admin-content', 'admin-notifications', 'admin-jobs', 'admin-api-clients', 'admin-webhooks', 'admin-audit-security', 'admin-operations'],
  },
]

const modules = new Map([
  ['platform-health', ['Platform health', 'platform_health']],
  ['public-app', ['App public capabilities', 'public_app']],
  ['public-content', ['Public content', 'public_content']],
  ['public-dictionary', ['Public dictionaries', 'public_dictionary']],
  ['api-client-auth', ['API Client authentication', 'api_client_auth']],
  ['api-client-notifications', ['Application notification submission', 'api_client_notifications']],
  ['mobile-auth', ['Authentication', 'mobile_auth']],
  ['mobile-profile', ['Profile and preferences', 'mobile_profile']],
  ['mobile-devices-sessions', ['Devices and sessions', 'mobile_devices_sessions']],
  ['mobile-notifications', ['Notifications', 'mobile_notifications']],
  ['mobile-security', ['Security events', 'mobile_security']],
  ['mobile-content', ['Bookmarks and legal consent', 'mobile_content']],
  ['admin-auth-profile', ['Authentication and profile', 'admin_auth_profile']],
  ['admin-dashboard', ['Dashboard', 'admin_dashboard']],
  ['admin-app-management', ['App management', 'admin_app_management']],
  ['admin-releases', ['Upgrade center', 'admin_releases']],
  ['admin-app-content', ['App content', 'admin_app_content']],
  ['admin-app-communications', ['App notifications', 'admin_app_communications']],
  ['admin-app-users', ['App users', 'admin_app_users']],
  ['admin-organization', ['Organization', 'admin_organization']],
  ['admin-tenants', ['Tenants', 'admin_tenants']],
  ['admin-users', ['Users', 'admin_users']],
  ['admin-access-control', ['Permissions', 'admin_access_control']],
  ['admin-system-settings', ['System settings', 'admin_system_settings']],
  ['admin-share-configuration', ['Share configuration', 'admin_share_configuration']],
  ['admin-storage', ['Files', 'admin_storage']],
  ['admin-content', ['Content', 'admin_content']],
  ['admin-notifications', ['Notifications', 'admin_notifications']],
  ['admin-jobs', ['Jobs', 'admin_jobs']],
  ['admin-api-clients', ['API Clients', 'admin_api_clients']],
  ['admin-webhooks', ['Webhooks', 'admin_webhooks']],
  ['admin-audit-security', ['Audit and security', 'admin_audit_security']],
  ['admin-operations', ['Online sessions and runtime', 'admin_operations']],
])

function expectedTag(path) {
  if (path.startsWith('/s/')) return 'public-content'
  if (path.startsWith('/internal/v1/health/')) return 'platform-health'
  if (path.startsWith('/api/v1/public/dictionaries/')) return 'public-dictionary'
  if (path === '/api/v1/articles' || path === '/api/v1/article-categories' || path.startsWith('/api/v1/articles/') || path.startsWith('/api/v1/article-assets/')) return 'public-content'
  if (path === '/api/v1/auth/client-token') return 'api-client-auth'
  if (path.startsWith('/api/v1/apps/') && path.includes('/notifications')) return 'api-client-notifications'
  if (path.startsWith('/api/v1/public/content/')) return 'public-content'
  if (path.startsWith('/api/v1/public/') || path === '/api/v1/regions') return 'public-app'
  if (path.startsWith('/api/v1/auth/')) return 'mobile-auth'
  if (path.startsWith('/api/v1/me/article-bookmarks') || path.startsWith('/api/v1/me/content-bookmarks') || path.startsWith('/api/v1/me/comments/') || path.startsWith('/api/v1/me/blocked-users/') || path.startsWith('/api/v1/content/items/') || path.startsWith('/api/v1/comments/') || path === '/api/v1/me/legal-consents') return 'mobile-content'
  if (path.startsWith('/api/v1/me/notification-preferences') || path.startsWith('/api/v1/me/notifications') || path.startsWith('/api/v1/me/push-devices') || path.startsWith('/api/v1/me/push-deliveries')) return 'mobile-notifications'
  if (path.startsWith('/api/v1/me/sessions') || path.startsWith('/api/v1/me/devices')) return 'mobile-devices-sessions'
  if (path === '/api/v1/me/login-events' || path === '/api/v1/me/security-events') return 'mobile-security'
  if (path === '/api/v1/me' || path === '/api/v1/me/preferences' || path.startsWith('/api/v1/me/account-deletion/')) return 'mobile-profile'
  if (path.startsWith('/admin-api/v1/apps/') && path.includes('/mobile/releases')) return 'admin-releases'
  if (path.startsWith('/admin-api/v1/mobile/releases')) return 'admin-releases'
  if (path.startsWith('/admin-api/v1/apps/') && path.includes('/content/')) return 'admin-app-content'
  if (path.startsWith('/admin-api/v1/apps/') && (path.includes('/notices') || path.includes('/messages'))) return 'admin-app-communications'
  if (path.startsWith('/admin-api/v1/apps/') && (path.includes('/notification-operations') || path.includes('/notification-runs') || path.includes('/notification-tasks') || path.includes('/notification-failures') || path.includes('/notification-retries'))) return 'admin-notifications'
  if (path.startsWith('/admin-api/v1/apps/') && path.includes('/users')) return 'admin-app-users'
  if (path.startsWith('/admin-api/v1/share-configs') || (path.startsWith('/admin-api/v1/apps/') && path.includes('/share-bindings'))) return 'admin-share-configuration'
  if (path === '/admin-api/v1/apps' || path.startsWith('/admin-api/v1/apps/')) return 'admin-app-management'
  if (path.startsWith('/admin-api/v1/auth/') || path === '/admin-api/v1/me' || path.startsWith('/admin-api/v1/me/')) return 'admin-auth-profile'
  if (path.startsWith('/admin-api/v1/dashboard/')) return 'admin-dashboard'
  if (path.startsWith('/admin-api/v1/org/')) return 'admin-organization'
  if (path.startsWith('/admin-api/v1/tenants')) return 'admin-tenants'
  if (path.startsWith('/admin-api/v1/users')) return 'admin-users'
  if (path.startsWith('/admin-api/v1/roles') || path === '/admin-api/v1/permissions' || path.startsWith('/admin-api/v1/menus')) return 'admin-access-control'
  if (['/admin-api/v1/configs', '/admin-api/v1/dict-types', '/admin-api/v1/dictionaries', '/admin-api/v1/dict-items', '/admin-api/v1/regions'].some((prefix) => path.startsWith(prefix))) return 'admin-system-settings'
  if (path.startsWith('/admin-api/v1/files')) return 'admin-storage'
  if (path.startsWith('/admin-api/v1/content/')) return 'admin-content'
  if (['/admin-api/v1/notices', '/admin-api/v1/messages', '/admin-api/v1/notification-templates', '/admin-api/v1/notification-deliveries'].some((prefix) => path.startsWith(prefix))) return 'admin-notifications'
  if (path === '/admin-api/v1/job-handlers' || path.startsWith('/admin-api/v1/job-schedules')) return 'admin-jobs'
  if (path.startsWith('/admin-api/v1/api-clients')) return 'admin-api-clients'
  if (path.startsWith('/admin-api/v1/webhooks')) return 'admin-webhooks'
  if (path.startsWith('/admin-api/v1/audit/') || path.startsWith('/admin-api/v1/block-rules')) return 'admin-audit-security'
  if (path.startsWith('/admin-api/v1/online-sessions') || path.startsWith('/admin-api/v1/ops/')) return 'admin-operations'
  return null
}

const errors = []
const source = await readFile(canonicalPath, 'utf8')
const document = parse(source)
if (typeof document?.openapi !== 'string' || !document.openapi.startsWith('3.1.')) errors.push('canonical document must use OpenAPI 3.1')

const declaredTags = Array.isArray(document?.tags) ? document.tags : []
const declaredNames = declaredTags.map((tag) => tag?.name)
if (JSON.stringify(declaredNames) !== JSON.stringify([...modules.keys()])) errors.push('top-level tags must match the registered module order')
for (const tag of declaredTags) {
  const metadata = modules.get(tag?.name)
  if (!metadata) continue
  const [displayName, keyCode] = metadata
  if (tag['x-displayName'] !== displayName) errors.push(`tag ${tag.name} has a drifting x-displayName`)
  if (tag['x-appkernia-i18n-key'] !== `api_reference.module.${keyCode}`) errors.push(`tag ${tag.name} has an invalid i18n key`)
}

const groups = Array.isArray(document?.['x-tagGroups']) ? document['x-tagGroups'] : []
if (groups.length !== surfaces.length) errors.push('x-tagGroups must declare exactly three interface surfaces')
const groupedTags = []
for (const [index, surface] of surfaces.entries()) {
  const group = groups[index]
  if (group?.name !== surface.name) errors.push(`surface ${surface.code} has a drifting canonical name`)
  if (group?.['x-appkernia-i18n-key'] !== `api_reference.surface.${surface.code}`) errors.push(`surface ${surface.code} has an invalid i18n key`)
  if (JSON.stringify(group?.tags) !== JSON.stringify(surface.tags)) errors.push(`surface ${surface.code} has an invalid module order`)
  groupedTags.push(...(Array.isArray(group?.tags) ? group.tags : []))
}
if (groupedTags.length !== new Set(groupedTags).size || JSON.stringify(groupedTags) !== JSON.stringify([...modules.keys()])) {
  errors.push('every registered module must belong to exactly one interface surface')
}

const pathOperations = []
const reusablePathItemOperations = []
const operationIds = new Set()
function validateOperation(operation, expected, location, target) {
  const operationId = operation?.operationId
  if (typeof operationId !== 'string' || operationId.length === 0 || operationIds.has(operationId)) {
    errors.push(`${location} has a missing or duplicate operationId`)
    return
  }
  operationIds.add(operationId)
  const tag = Array.isArray(operation.tags) && operation.tags.length === 1 ? operation.tags[0] : null
  if (!expected) errors.push(`${location} is not covered by the module mapping`)
  if (tag !== expected) errors.push(`${operationId} must use exactly the registered module ${expected ?? '(unmapped)'}`)
  if (typeof operation.summary !== 'string' || operation.summary.length === 0) errors.push(`${operationId} has no canonical English summary`)
  target.push({ operationId, summary: operation.summary })
}
for (const [path, pathItem] of Object.entries(document?.paths ?? {})) {
  for (const [method, operation] of Object.entries(pathItem ?? {})) {
    if (!operationMethods.has(method)) continue
    const expected = expectedTag(path)
    validateOperation(operation, expected, `${method.toUpperCase()} ${path}`, pathOperations)
  }
}
for (const [name, pathItem] of Object.entries(document?.components?.pathItems ?? {})) {
  for (const [method, operation] of Object.entries(pathItem ?? {})) {
    if (!operationMethods.has(method)) continue
    validateOperation(operation, 'admin-users', `components.pathItems.${name}.${method}`, reusablePathItemOperations)
  }
}
const operations = [...pathOperations, ...reusablePathItemOperations]

const catalogs = {}
for (const locale of ['zh-CN', 'en-US']) {
  catalogs[locale] = JSON.parse(await readFile(resolve(root, `blueprint/i18n/admin/${locale}.json`), 'utf8'))
}
const expectedKeys = new Set([
  ...surfaces.map((surface) => `api_reference.surface.${surface.code}`),
  ...[...modules.values()].map(([, keyCode]) => `api_reference.module.${keyCode}`),
  ...operations.map(({ operationId }) => `api_reference.operation.${operationId}`),
])
for (const locale of ['zh-CN', 'en-US']) {
  const catalog = catalogs[locale]
  const actualKeys = Object.keys(catalog).filter((key) => key.startsWith('api_reference.'))
  const missing = [...expectedKeys].filter((key) => typeof catalog[key] !== 'string' || catalog[key].trim().length === 0)
  const stale = actualKeys.filter((key) => !expectedKeys.has(key))
  if (missing.length > 0) errors.push(`${locale} is missing api_reference keys: ${missing.join(', ')}`)
  if (stale.length > 0) errors.push(`${locale} has stale api_reference keys: ${stale.join(', ')}`)
}
for (const { operationId, summary } of operations) {
  const key = `api_reference.operation.${operationId}`
  if (catalogs['en-US'][key] !== summary) errors.push(`${key} must exactly match the canonical English summary`)
}
for (const surface of surfaces) {
  if (catalogs['en-US'][`api_reference.surface.${surface.code}`] !== surface.name) errors.push(`English surface ${surface.code} drifted from canonical metadata`)
}
for (const [tag, [displayName, keyCode]] of modules) {
  if (catalogs['en-US'][`api_reference.module.${keyCode}`] !== displayName) errors.push(`English module ${tag} drifted from canonical metadata`)
}

const report = {
  generated_at: new Date().toISOString(),
  passed: errors.length === 0,
  counts: {
    interface_surfaces: surfaces.length,
    modules: modules.size,
    path_operations: pathOperations.length,
    reusable_path_item_operations: reusablePathItemOperations.length,
    localized_operation_titles: operations.length,
    operations_with_one_registered_tag: operations.filter(({ operationId }) => !errors.some((error) => error.startsWith(operationId))).length,
    translation_keys_per_locale: expectedKeys.size,
  },
  errors,
}
await mkdir(resolve(root, 'output'), { recursive: true })
await writeFile(resolve(root, 'output/openapi-reference-contract.json'), `${JSON.stringify(report, null, 2)}\n`, 'utf8')
console.log(JSON.stringify(report, null, 2))
if (!report.passed) process.exitCode = 1
