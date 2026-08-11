import { gzipSync } from 'node:zlib'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '../../..')
const dist = resolve(import.meta.dirname, '../dist')
const manifest = JSON.parse(await readFile(resolve(dist, '.vite/manifest.json'), 'utf8'))
const initialBudget = 300 * 1024
const chunkBudget = 180 * 1024
const openApiInitialBudget = 1100 * 1024
const openApiChunkBudget = 750 * 1024
const entries = Object.entries(manifest)
const adminEntry = entries.find(([key, value]) => key === 'index.html' && value.isEntry)
const openApiEntry = entries.find(([key, value]) => key === 'openapi/index.html' && value.isEntry)
if (!adminEntry) throw new Error('Vite manifest has no Admin application entry')
if (!openApiEntry) throw new Error('Vite manifest has no OpenAPI documentation entry')

function collectKeys(entryKey, includeDynamicImports) {
  const keys = new Set()
  function collect(key) {
    if (keys.has(key)) return
    keys.add(key)
    for (const imported of manifest[key]?.imports ?? []) collect(imported)
    if (includeDynamicImports) {
      for (const imported of manifest[key]?.dynamicImports ?? []) collect(imported)
    }
  }
  collect(entryKey)
  return keys
}

const adminInitialKeys = collectKeys(adminEntry[0], false)
const adminAllKeys = collectKeys(adminEntry[0], true)
const openApiInitialKeys = collectKeys(openApiEntry[0], false)
const openApiAllKeys = collectKeys(openApiEntry[0], true)
const adminScalarKeys = [...adminAllKeys].filter((key) => key.includes('@scalar/'))
const docsOnlyMarkers = ['api_reference.operation.getLiveness', 'OPENAPI_DOCUMENT_VERSION_UNSUPPORTED']

const gzipCache = new Map()
async function gzipBytes(file) {
  if (!file.endsWith('.js') && !file.endsWith('.css')) return 0
  if (!gzipCache.has(file)) {
    const bytes = await readFile(resolve(dist, file))
    gzipCache.set(file, gzipSync(bytes, { level: 9 }).byteLength)
  }
  return gzipCache.get(file)
}

async function resourcesFor(keys) {
  const files = new Set()
  for (const key of keys) {
    const entry = manifest[key]
    if (typeof entry?.file === 'string' && (entry.file.endsWith('.js') || entry.file.endsWith('.css'))) files.add(entry.file)
    for (const css of entry?.css ?? []) files.add(css)
  }
  const resources = []
  for (const file of files) resources.push({ file, gzip_bytes: await gzipBytes(file) })
  return resources.sort((left, right) => right.gzip_bytes - left.gzip_bytes)
}

async function markerMatches(resources, markers) {
  const matches = []
  for (const resource of resources) {
    if (!resource.file.endsWith('.js')) continue
    const source = await readFile(resolve(dist, resource.file), 'utf8')
    for (const marker of markers) {
      if (source.includes(marker)) matches.push({ file: resource.file, marker })
    }
  }
  return matches
}

const initialFiles = await resourcesFor(adminInitialKeys)
const chunks = (await resourcesFor(adminAllKeys)).filter((resource) => resource.file.endsWith('.js'))
const openApiInitialFiles = await resourcesFor(openApiInitialKeys)
const openApiChunks = (await resourcesFor(openApiAllKeys)).filter((resource) => resource.file.endsWith('.js'))
const initialGzipBytes = initialFiles.reduce((total, item) => total + item.gzip_bytes, 0)
const openApiInitialGzipBytes = openApiInitialFiles.reduce((total, item) => total + item.gzip_bytes, 0)
const oversizedChunks = chunks.filter((chunk) => chunk.gzip_bytes > chunkBudget)
const oversizedOpenApiChunks = openApiChunks.filter((chunk) => chunk.gzip_bytes > openApiChunkBudget)
const adminDocsOnlyMatches = await markerMatches(await resourcesFor(adminAllKeys), docsOnlyMarkers)
const openApiDocsOnlyMatches = await markerMatches(await resourcesFor(openApiAllKeys), docsOnlyMarkers)
const report = {
  generated_at: new Date().toISOString(),
  budgets: {
    initial_gzip_bytes: initialBudget,
    chunk_gzip_bytes: chunkBudget,
    openapi_initial_gzip_bytes: openApiInitialBudget,
    openapi_chunk_gzip_bytes: openApiChunkBudget,
  },
  actual: {
    initial_gzip_bytes: initialGzipBytes,
    largest_chunks: chunks.slice(0, 10),
    openapi_initial_gzip_bytes: openApiInitialGzipBytes,
    openapi_largest_chunks: openApiChunks.slice(0, 10),
  },
  initial_files: initialFiles.sort((left, right) => right.gzip_bytes - left.gzip_bytes),
  openapi_initial_files: openApiInitialFiles,
  passed: initialGzipBytes <= initialBudget
    && oversizedChunks.length === 0
    && openApiInitialGzipBytes <= openApiInitialBudget
    && oversizedOpenApiChunks.length === 0
    && adminScalarKeys.length === 0
    && adminDocsOnlyMatches.length === 0
    && new Set(openApiDocsOnlyMatches.map(({ marker }) => marker)).size === docsOnlyMarkers.length,
  admin_docs_only_matches: adminDocsOnlyMatches,
  admin_scalar_keys: adminScalarKeys,
  openapi_docs_only_matches: openApiDocsOnlyMatches,
  oversized_chunks: oversizedChunks,
  openapi_oversized_chunks: oversizedOpenApiChunks,
}

await mkdir(resolve(root, 'output'), { recursive: true })
await writeFile(resolve(root, 'output/admin-bundle-budget.json'), `${JSON.stringify(report, null, 2)}\n`, 'utf8')
console.log(JSON.stringify(report, null, 2))
if (!report.passed) process.exitCode = 1
