import { readFile, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'

const sourceUrl = new URL('../../../blueprint/admin-frontend/spec/admin-route-registry.json', import.meta.url)
const targetUrl = new URL('../src/generated/route-registry.ts', import.meta.url)
const registry = JSON.parse(await readFile(sourceUrl, 'utf8'))

const routes = registry.routes.map((route) => ({
  componentKey: route.component_key,
  path: route.path,
  auth: route.auth,
  layout: route.layout,
  titleKey: route.title_key,
  permissions: route.permissions ?? [],
  featureFlag: route.feature_flag ?? null,
  activeMenuCode: route.active_menu_code ?? null,
}))

const output = `// This file is generated from blueprint/admin-frontend/spec/admin-route-registry.json.\n` +
  `// Do not edit manually.\n\n` +
  `export const generatedRouteRegistry = ${JSON.stringify(routes, null, 2)} as const\n`

await writeFile(targetUrl, output)
console.log(`generated ${routes.length} admin routes`)
