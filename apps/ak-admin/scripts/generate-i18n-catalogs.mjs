import { mkdir, readFile, writeFile } from 'node:fs/promises'

const locales = ['zh-CN', 'en-US']
const namespaces = ['common', 'auth', 'navigation', 'validation', 'errors', 'profile', 'settings', 'system', 'notifications', 'content', 'mobile_releases', 'apps', 'share_configs', 'push_channels', 'login_providers', 'openapi', 'api_reference']

function namespaceFor(key) {
  const prefix = key.split('.', 1)[0]
  if (['app', 'common', 'meta'].includes(prefix)) return 'common'
  if (['menu', 'routes', 'shell', 'dashboard'].includes(prefix)) return 'navigation'
  if (['schedules', 'api_clients', 'webhooks', 'block_rules', 'ops'].includes(prefix)) return 'system'
  if (prefix === 'notification_operations') return 'notifications'
  if (namespaces.includes(prefix)) return prefix
  throw new Error(`unmapped Admin translation key: ${key}`)
}

for (const locale of locales) {
  const source = JSON.parse(await readFile(new URL(`../../../blueprint/i18n/admin/${locale}.json`, import.meta.url), 'utf8'))
  const catalogs = Object.fromEntries(namespaces.map((namespace) => [namespace, {}]))
  for (const [key, value] of Object.entries(source)) catalogs[namespaceFor(key)][key] = value
  const target = new URL(`../src/locales/${locale}/`, import.meta.url)
  await mkdir(target, { recursive: true })
  for (const namespace of namespaces) {
    await writeFile(new URL(`${namespace}.json`, target), `${JSON.stringify(catalogs[namespace], null, 2)}\n`)
  }
}

console.log(`generated ${locales.length * namespaces.length} Admin i18n namespace catalogs`)
