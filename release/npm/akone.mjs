#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const key = `${process.platform}-${process.arch}`
const packages = {
  'darwin-arm64': ['@appkernia/akone-darwin-arm64', 'akone'],
  'darwin-x64': ['@appkernia/akone-darwin-x64', 'akone'],
  'linux-arm64': ['@appkernia/akone-linux-arm64', 'akone'],
  'linux-x64': ['@appkernia/akone-linux-x64', 'akone'],
  'win32-x64': ['@appkernia/akone-win32-x64', 'akone.exe'],
}

const selected = packages[key]
if (!selected) {
  process.stderr.write(`akone: unsupported npm platform ${process.platform}/${process.arch}\n`)
  process.exit(1)
}

let binary
try {
  binary = require.resolve(`${selected[0]}/bin/${selected[1]}`)
} catch {
  process.stderr.write(`akone: platform package ${selected[0]} is missing; reinstall @appkernia/akone\n`)
  process.exit(1)
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit', windowsHide: false })
if (result.error) {
  process.stderr.write(`akone: ${result.error.message}\n`)
  process.exit(1)
}
process.exit(result.status ?? 1)
