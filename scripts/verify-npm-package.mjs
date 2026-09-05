#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { dirname, isAbsolute, relative, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { normalizeVersion } from './release.mjs'

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const SRI_PATTERN = /^sha512-[A-Za-z0-9+/]+={0,2}$/

function npm(args) {
  const result = spawnSync('npm', args, {
    cwd: ROOT,
    encoding: 'utf8',
    env: process.env,
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  if (result.error) throw result.error
  return result
}

function validIntegrity(value) {
  return typeof value === 'string' && SRI_PATTERN.test(value)
}

export function parsePackedIntegrity(output) {
  let packages
  try {
    packages = JSON.parse(output)
  } catch {
    throw new Error('npm pack did not return JSON')
  }
  const integrity = Array.isArray(packages) && packages.length === 1 ? packages[0]?.integrity : undefined
  if (!validIntegrity(integrity)) throw new Error('npm pack did not return one sha512 integrity')
  return integrity
}

export function parsePublishedIntegrity(output) {
  let integrity
  try {
    integrity = JSON.parse(output)
  } catch {
    throw new Error('npm view did not return JSON')
  }
  if (!validIntegrity(integrity)) throw new Error('published npm package has no valid sha512 integrity')
  return integrity
}

export function isMissingPackage(result) {
  if (result.status === 0) return false
  return /(?:^|\W)E404(?:\W|$)|404 Not Found/i.test(`${result.stdout || ''}\n${result.stderr || ''}`)
}

export function assertMatchingIntegrity(packageName, version, localIntegrity, publishedIntegrity) {
  if (localIntegrity !== publishedIntegrity) {
    throw new Error(`${packageName}@${version} already exists with different content; refusing to skip it`)
  }
}

export function verifyNpmPackage(packageDirectory, expectedVersion) {
  const candidate = resolve(packageDirectory)
  const child = relative(ROOT, candidate)
  if (!child || child.startsWith('..') || isAbsolute(child)) {
    throw new Error(`npm package directory must be below ${ROOT}`)
  }

  const packageJSON = JSON.parse(readFileSync(resolve(candidate, 'package.json'), 'utf8'))
  const version = normalizeVersion(expectedVersion)
  if (packageJSON.version !== version || typeof packageJSON.name !== 'string' || !packageJSON.name) {
    throw new Error('staged npm package name/version does not match the release')
  }

  const published = npm(['view', `${packageJSON.name}@${version}`, 'dist.integrity', '--json'])
  if (published.status !== 0) {
    if (isMissingPackage(published)) return 'missing'
    throw new Error(`npm view ${packageJSON.name}@${version} failed with exit ${published.status}`)
  }

  const packed = npm(['pack', candidate, '--dry-run', '--json', '--ignore-scripts'])
  if (packed.status !== 0) throw new Error(`npm pack ${packageJSON.name}@${version} failed with exit ${packed.status}`)
  const localIntegrity = parsePackedIntegrity(packed.stdout)
  const publishedIntegrity = parsePublishedIntegrity(published.stdout)
  assertMatchingIntegrity(packageJSON.name, version, localIntegrity, publishedIntegrity)
  return 'matching'
}

function parseArgs(argv) {
  const options = {}
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index]
    if (argument === '--package-dir' || argument === '--version') {
      options[argument.slice(2).replace('-', '')] = argv[index + 1]
      index += 1
    } else {
      throw new Error(`unknown argument: ${argument}`)
    }
  }
  if (!options.packagedir || !options.version) throw new Error('--package-dir and --version are required')
  return options
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try {
    const options = parseArgs(process.argv.slice(2))
    process.stdout.write(`${verifyNpmPackage(options.packagedir, options.version)}\n`)
  } catch (error) {
    process.stderr.write(`release.npm.verify.error=${error instanceof Error ? error.message : String(error)}\n`)
    process.exitCode = 1
  }
}
