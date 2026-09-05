#!/usr/bin/env node

import { cp, lstat, mkdir, readdir, rm } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const DEFAULT_SOURCE = join(ROOT, 'apps/ak-admin/dist')
const DEFAULT_DESTINATION = join(ROOT, 'server/internal/platform/adminui/dist')

function excluded(relativePath) {
  return relativePath.endsWith('.map') || relativePath.split('/').includes('.vite')
}

export async function stageAdmin(source = DEFAULT_SOURCE, destination = DEFAULT_DESTINATION) {
  const index = await lstat(join(source, 'index.html')).catch(() => null)
  if (!index?.isFile()) throw new Error(`Admin build is missing ${join(source, 'index.html')}`)

  await rm(destination, { force: true, recursive: true })
  await mkdir(destination, { recursive: true })
  let files = 0

  async function copyDirectory(currentSource, currentDestination, relative = '') {
    const entries = await readdir(currentSource, { withFileTypes: true })
    for (const entry of entries) {
      const childRelative = relative ? `${relative}/${entry.name}` : entry.name
      if (excluded(childRelative)) continue
      const sourcePath = join(currentSource, entry.name)
      const destinationPath = join(currentDestination, entry.name)
      if (entry.isSymbolicLink()) throw new Error(`Admin build contains a symbolic link: ${childRelative}`)
      if (entry.isDirectory()) {
        await mkdir(destinationPath, { recursive: true })
        await copyDirectory(sourcePath, destinationPath, childRelative)
      } else if (entry.isFile()) {
        await cp(sourcePath, destinationPath)
        files += 1
      } else {
        throw new Error(`Admin build contains an unsupported file: ${childRelative}`)
      }
    }
  }

  await copyDirectory(source, destination)
  if (files === 0) throw new Error('Admin staging produced no files')
  return { destination, files }
}

async function main() {
  const result = await stageAdmin()
  process.stdout.write(`admin.stage.files=${result.files}\nadmin.stage.dir=${result.destination}\n`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch((error) => {
    process.stderr.write(`admin.stage.error=${error instanceof Error ? error.message : String(error)}\n`)
    process.exitCode = 1
  })
}
