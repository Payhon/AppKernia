import { gzipSync } from 'node:zlib'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '../../..')
const dist = resolve(import.meta.dirname, '../dist')
const manifest = JSON.parse(await readFile(resolve(dist, '.vite/manifest.json'), 'utf8'))
const initialBudget = 300 * 1024
const chunkBudget = 180 * 1024
const entries = Object.entries(manifest)
const entry = entries.find(([, value]) => value.isEntry)
if (!entry) throw new Error('Vite manifest has no application entry')

const initialKeys = new Set()
function collectInitial(key) {
  if (initialKeys.has(key)) return
  initialKeys.add(key)
  for (const imported of manifest[key]?.imports ?? []) collectInitial(imported)
}
collectInitial(entry[0])

const gzipCache = new Map()
async function gzipBytes(file) {
  if (!file.endsWith('.js')) return 0
  if (!gzipCache.has(file)) {
    const bytes = await readFile(resolve(dist, file))
    gzipCache.set(file, gzipSync(bytes, { level: 9 }).byteLength)
  }
  return gzipCache.get(file)
}

const initialFiles = []
for (const key of initialKeys) {
  const file = manifest[key]?.file
  if (typeof file === 'string' && file.endsWith('.js')) initialFiles.push({ file, gzip_bytes: await gzipBytes(file) })
}
const initialGzipBytes = initialFiles.reduce((total, item) => total + item.gzip_bytes, 0)

const chunks = []
for (const [, value] of entries) {
  if (typeof value.file !== 'string' || !value.file.endsWith('.js')) continue
  chunks.push({ file: value.file, gzip_bytes: await gzipBytes(value.file) })
}
chunks.sort((left, right) => right.gzip_bytes - left.gzip_bytes)
const oversizedChunks = chunks.filter((chunk) => chunk.gzip_bytes > chunkBudget)
const report = {
  generated_at: new Date().toISOString(),
  budgets: { initial_gzip_bytes: initialBudget, chunk_gzip_bytes: chunkBudget },
  actual: { initial_gzip_bytes: initialGzipBytes, largest_chunks: chunks.slice(0, 10) },
  initial_files: initialFiles.sort((left, right) => right.gzip_bytes - left.gzip_bytes),
  passed: initialGzipBytes <= initialBudget && oversizedChunks.length === 0,
  oversized_chunks: oversizedChunks,
}

await mkdir(resolve(root, 'output'), { recursive: true })
await writeFile(resolve(root, 'output/admin-bundle-budget.json'), `${JSON.stringify(report, null, 2)}\n`, 'utf8')
console.log(JSON.stringify(report, null, 2))
if (!report.passed) process.exitCode = 1
