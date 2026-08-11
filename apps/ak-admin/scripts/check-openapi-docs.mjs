import { createHash } from 'node:crypto'
import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '../../..')
const adminRoot = resolve(import.meta.dirname, '..')
const dist = resolve(adminRoot, 'dist')
const canonicalPath = resolve(root, 'server/openapi/openapi.yaml')
const emittedPath = resolve(dist, 'openapi/openapi.yaml')
const htmlPath = resolve(dist, 'openapi/index.html')
const manifestPath = resolve(dist, '.vite/manifest.json')

const [canonical, emitted, html, manifestSource, openApiDirectory] = await Promise.all([
  readFile(canonicalPath),
  readFile(emittedPath),
  readFile(htmlPath, 'utf8'),
  readFile(manifestPath, 'utf8'),
  readdir(resolve(dist, 'openapi')),
])
const manifest = JSON.parse(manifestSource)
const adminEntry = manifest['index.html']
const openApiEntry = manifest['openapi/index.html']
const canonicalSha256 = createHash('sha256').update(canonical).digest('hex')
const emittedSha256 = createHash('sha256').update(emitted).digest('hex')

const checks = {
  admin_entry_present: adminEntry?.isEntry === true,
  canonical_openapi_31: canonical.toString('utf8', 0, 64).includes('openapi: 3.1.'),
  docs_entry_is_independent: openApiEntry?.isEntry === true && openApiEntry.file !== adminEntry?.file,
  docs_entry_present: openApiEntry?.isEntry === true,
  emitted_spec_is_byte_identical: Buffer.compare(canonical, emitted) === 0,
  html_contains_no_remote_asset: !/\b(?:https?:)?\/\//i.test(html),
  html_uses_bundled_entry: typeof openApiEntry?.file === 'string' && html.includes(`/${openApiEntry.file}`),
  no_locale_specific_spec: openApiDirectory.sort().join(',') === 'index.html,openapi.yaml',
}
const passed = Object.values(checks).every(Boolean)
const report = {
  generated_at: new Date().toISOString(),
  passed,
  checks,
  canonical: {
    bytes: canonical.byteLength,
    path: 'server/openapi/openapi.yaml',
    sha256: canonicalSha256,
  },
  emitted: {
    bytes: emitted.byteLength,
    path: 'apps/ak-admin/dist/openapi/openapi.yaml',
    sha256: emittedSha256,
  },
  entries: {
    admin: adminEntry?.file ?? null,
    openapi: openApiEntry?.file ?? null,
  },
}

await mkdir(resolve(root, 'output'), { recursive: true })
await writeFile(resolve(root, 'output/admin-openapi-docs.json'), `${JSON.stringify(report, null, 2)}\n`, 'utf8')
console.log(JSON.stringify(report, null, 2))
if (!passed) process.exitCode = 1
