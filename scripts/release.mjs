#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { dirname, resolve } from 'node:path'

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const GORELEASER_VERSION = 'v2.17.1'
const RELEASE_REMOTES = ['gitee', 'origin']

export function normalizeVersion(input) {
  const value = String(input ?? '').trim().replace(/^v/, '')
  const semver = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/
  const match = semver.exec(value)
  const invalidNumericPrerelease = match?.[4]
    ?.split('.')
    .some((identifier) => /^\d+$/.test(identifier) && identifier.length > 1 && identifier.startsWith('0'))
  if (!match || invalidNumericPrerelease) {
    throw new Error(`invalid release version: ${input ?? ''}`)
  }
  return value
}

export function isStableVersion(version) {
  return !version.includes('-')
}

export function parseArgs(argv, env = process.env) {
  const result = { publish: false, version: env.VERSION }
  let mode
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index]
    if (argument === '--publish') {
      if (mode === 'dry-run') throw new Error('--publish and --dry-run cannot be combined')
      mode = 'publish'
      result.publish = true
    } else if (argument === '--dry-run') {
      if (mode === 'publish') throw new Error('--publish and --dry-run cannot be combined')
      mode = 'dry-run'
      result.publish = false
    } else if (argument === '--version') {
      result.version = argv[index + 1]
      index += 1
    } else if (argument === '--help' || argument === '-h') {
      result.help = true
    } else {
      throw new Error(`unknown argument: ${argument}`)
    }
  }
  if (!result.help) result.version = normalizeVersion(result.version)
  return result
}

function printable(command, args) {
  return [command, ...args]
    .map((part) => (/^[A-Za-z0-9_./:=@+-]+$/.test(part) ? part : JSON.stringify(part)))
    .join(' ')
}

function run(command, args, options = {}) {
  process.stdout.write(`+ ${printable(command, args)}\n`)
  const result = spawnSync(command, args, {
    cwd: ROOT,
    env: process.env,
    stdio: options.capture ? ['ignore', 'pipe', 'pipe'] : 'inherit',
    encoding: options.capture ? 'utf8' : undefined,
  })
  if (result.error) throw result.error
  if (result.status !== 0 && !options.allowFailure) {
    const detail = options.capture ? String(result.stderr || result.stdout || '').trim() : ''
    throw new Error(`${printable(command, args)} failed with exit ${result.status}${detail ? `: ${detail}` : ''}`)
  }
  return result
}

function capture(command, args) {
  return String(run(command, args, { capture: true }).stdout).trim()
}

function assertCleanTree() {
  const status = capture('git', ['status', '--porcelain=v1', '--untracked-files=all'])
  if (status) throw new Error(`working tree is not clean:\n${status}`)
}

function assertReleasePolicy(version) {
  if (isStableVersion(version)) {
    throw new Error(
      'stable akone releases are blocked until macOS notarization and Windows Authenticode signing are implemented; use an explicit prerelease version such as 1.0.0-preview.1',
    )
  }
}

function fetchAndAssertHeads(head) {
  for (const remote of RELEASE_REMOTES) {
    run('git', ['fetch', '--no-tags', remote, '+refs/heads/main:refs/remotes/' + remote + '/main'])
    const remoteHead = capture('git', ['rev-parse', `refs/remotes/${remote}/main`])
    if (remoteHead !== head) {
      throw new Error(`HEAD ${head} does not match ${remote}/main ${remoteHead}`)
    }
  }
}

function localTagState(tag, head) {
  const exists = run('git', ['show-ref', '--verify', '--quiet', `refs/tags/${tag}`], {
    capture: true,
    allowFailure: true,
  }).status === 0
  if (!exists) return 'absent'
  run('git', ['verify-tag', tag])
  const commit = capture('git', ['rev-list', '-n', '1', tag])
  if (commit !== head) throw new Error(`local ${tag} points to ${commit}, expected ${head}`)
  return 'matching'
}

function remoteTagState(remote, tag, head) {
  const output = capture('git', [
    'ls-remote',
    '--tags',
    remote,
    `refs/tags/${tag}`,
    `refs/tags/${tag}^{}`,
  ])
  if (!output) return 'absent'
  const lines = output.split('\n').filter(Boolean)
  const peeled = lines.find((line) => line.endsWith(`refs/tags/${tag}^{}`))
  if (!peeled) throw new Error(`${remote}/${tag} exists but is not a signed annotated tag`)
  const commit = peeled.split(/\s+/)[0]
  if (commit !== head) throw new Error(`${remote}/${tag} points to ${commit}, expected ${head}`)
  return 'matching'
}

function assertNoTrackedSecrets() {
  const tracked = capture('git', ['ls-files', '--', '.env*', '.secrets/**'])
    .split('\n')
    .filter((path) => path && path !== '.env.example')
  if (tracked.length) throw new Error(`release refuses tracked secret paths:\n${tracked.join('\n')}`)
}

function runQualityGates() {
  run('make', ['check'])
  run('node', ['--test', 'scripts/release.test.mjs'])
  run('go', [
    'run',
    `github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION}`,
    'check',
    '--config',
    '.goreleaser.yaml',
  ])
  run('go', [
    'run',
    `github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION}`,
    'build',
    '--snapshot',
    '--clean',
    '--config',
    '.goreleaser.yaml',
  ])
}

function printHelp() {
  process.stdout.write(`Usage: node scripts/release.mjs --version X.Y.Z-preview.N [--publish]\n\n`)
  process.stdout.write('Without --publish this runs the complete release preflight and does not create or push a tag.\n')
}

export function main(argv = process.argv.slice(2)) {
  const options = parseArgs(argv)
  if (options.help) {
    printHelp()
    return
  }

  const { version, publish } = options
  const tag = `v${version}`
  assertReleasePolicy(version)

  const repositoryRoot = capture('git', ['rev-parse', '--show-toplevel'])
  if (resolve(repositoryRoot) !== ROOT) throw new Error(`run release from ${ROOT}`)
  if (capture('git', ['branch', '--show-current']) !== 'main') throw new Error('release must run from main')
  assertCleanTree()
  assertNoTrackedSecrets()

  for (const remote of RELEASE_REMOTES) capture('git', ['remote', 'get-url', remote])
  const head = capture('git', ['rev-parse', 'HEAD'])
  fetchAndAssertHeads(head)

  const localState = localTagState(tag, head)
  const remoteStates = Object.fromEntries(RELEASE_REMOTES.map((remote) => [remote, remoteTagState(remote, tag, head)]))
  if (!publish && (localState !== 'absent' || Object.values(remoteStates).some((state) => state !== 'absent'))) {
    throw new Error(`${tag} already exists; dry-run release versions must be new`)
  }
  if (publish && localState === 'absent' && Object.values(remoteStates).some((state) => state !== 'absent')) {
    throw new Error(`${tag} exists remotely without the matching local signed tag; refusing an ambiguous retry`)
  }

  runQualityGates()
  assertCleanTree()
  fetchAndAssertHeads(head)

  if (!publish) {
    process.stdout.write(`release.preflight=passed\nrelease.tag=${tag}\nrelease.mode=dry-run\n`)
    return
  }

  const signingKey = run('git', ['config', '--get', 'user.signingkey'], {
    capture: true,
    allowFailure: true,
  })
  if (signingKey.status !== 0 || !String(signingKey.stdout).trim()) {
    throw new Error('git user.signingkey is required for --publish')
  }
  if (localState === 'absent') {
    run('git', ['tag', '-s', '-a', tag, '-m', `AppKernia akone ${tag}`])
    run('git', ['verify-tag', tag])
  }

  for (const remote of RELEASE_REMOTES) {
    if (remoteStates[remote] === 'absent') run('git', ['push', remote, `refs/tags/${tag}`])
    if (remoteTagState(remote, tag, head) !== 'matching') throw new Error(`failed to verify ${remote}/${tag}`)
  }
  process.stdout.write(`release.tag=${tag}\nrelease.mode=published-tag\n`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try {
    main()
  } catch (error) {
    process.stderr.write(`release.error=${error instanceof Error ? error.message : String(error)}\n`)
    process.exitCode = 1
  }
}
