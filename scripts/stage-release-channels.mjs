#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { chmod, copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, isAbsolute, join, relative, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { normalizeVersion } from './release.mjs'

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const DEFAULT_ASSETS = join(ROOT, 'release-assets')
const DEFAULT_OUTPUT = join(ROOT, 'dist/channels')

export const NPM_PLATFORMS = [
  { goos: 'darwin', goarch: 'amd64', npmOs: 'darwin', npmCpu: 'x64' },
  { goos: 'darwin', goarch: 'arm64', npmOs: 'darwin', npmCpu: 'arm64' },
  { goos: 'linux', goarch: 'amd64', npmOs: 'linux', npmCpu: 'x64' },
  { goos: 'linux', goarch: 'arm64', npmOs: 'linux', npmCpu: 'arm64' },
  { goos: 'windows', goarch: 'amd64', npmOs: 'win32', npmCpu: 'x64' },
]

export function npmPackageDefinitions(version) {
  version = normalizeVersion(version)
  return NPM_PLATFORMS.map((platform) => ({
    ...platform,
    archiveName: `akone_${version}_${platform.goos}_${platform.goarch}.${platform.goos === 'windows' ? 'zip' : 'tar.gz'}`,
    packageName: `@appkernia/akone-${platform.npmOs}-${platform.npmCpu}`,
    version,
  }))
}

function run(command, args) {
  const result = spawnSync(command, args, { cwd: ROOT, stdio: 'pipe', encoding: 'utf8' })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`${command} ${args.join(' ')} failed: ${String(result.stderr).trim()}`)
}

function assertSafeOutput(output) {
  const candidate = resolve(output)
  const allowed = [ROOT, resolve(tmpdir())].some((parent) => {
    const child = relative(parent, candidate)
    return child && !child.startsWith('..') && !isAbsolute(child)
  })
  if (!allowed) throw new Error(`release output must be below ${ROOT} or the system temporary directory`)
}

function packageJson(name, version, extra = {}) {
  return {
    name,
    version,
    description: 'AppKernia all-in-one server and agent-friendly CLI',
    license: 'MIT',
    repository: {
      type: 'git',
      url: 'git+https://github.com/Payhon/AppKernia.git',
    },
    engines: { node: '>=18' },
    publishConfig: { access: 'public', provenance: true },
    ...extra,
  }
}

async function writeJson(path, value) {
  await writeFile(path, `${JSON.stringify(value, null, 2)}\n`)
}

async function extractBinary(archive, output, windows) {
  const directory = await mkdtemp(join(tmpdir(), 'akone-npm-'))
  try {
    const filename = windows ? 'akone.exe' : 'akone'
    if (windows) run('unzip', ['-qq', archive, filename, '-d', directory])
    else run('tar', ['-xzf', archive, '-C', directory, filename])
    await copyFile(join(directory, filename), output)
    if (!windows) await chmod(output, 0o755)
  } finally {
    await rm(directory, { force: true, recursive: true })
  }
}

export async function stageNpm({ version, assets = DEFAULT_ASSETS, output = DEFAULT_OUTPUT }) {
  version = normalizeVersion(version)
  assertSafeOutput(output)
  const npmOutput = join(output, 'npm')
  await rm(npmOutput, { force: true, recursive: true })
  await mkdir(npmOutput, { recursive: true })

  const optionalDependencies = {}
  const publishOrder = []
  for (const platform of npmPackageDefinitions(version)) {
    const windows = platform.goos === 'windows'
    const packageDirectory = join(npmOutput, `${platform.npmOs}-${platform.npmCpu}`)
    const binaryDirectory = join(packageDirectory, 'bin')
    const binaryName = windows ? 'akone.exe' : 'akone'
    await mkdir(binaryDirectory, { recursive: true })
    await extractBinary(join(assets, platform.archiveName), join(binaryDirectory, binaryName), windows)
    await copyFile(join(ROOT, 'LICENSE'), join(packageDirectory, 'LICENSE'))
    await copyFile(join(ROOT, 'THIRD_PARTY_NOTICES.md'), join(packageDirectory, 'THIRD_PARTY_NOTICES.md'))
    await writeJson(
      join(packageDirectory, 'package.json'),
      packageJson(platform.packageName, version, {
        os: [platform.npmOs],
        cpu: [platform.npmCpu],
        files: ['bin', 'LICENSE', 'THIRD_PARTY_NOTICES.md'],
      }),
    )
    optionalDependencies[platform.packageName] = version
    publishOrder.push(relative(ROOT, packageDirectory))
  }

  const metaDirectory = join(npmOutput, 'akone')
  await mkdir(join(metaDirectory, 'bin'), { recursive: true })
  await copyFile(join(ROOT, 'release/npm/akone.mjs'), join(metaDirectory, 'bin/akone.mjs'))
  await copyFile(join(ROOT, 'release/npm/README.md'), join(metaDirectory, 'README.md'))
  await copyFile(join(ROOT, 'LICENSE'), join(metaDirectory, 'LICENSE'))
  await copyFile(join(ROOT, 'THIRD_PARTY_NOTICES.md'), join(metaDirectory, 'THIRD_PARTY_NOTICES.md'))
  await writeJson(
    join(metaDirectory, 'package.json'),
    packageJson('@appkernia/akone', version, {
      bin: { akone: 'bin/akone.mjs' },
      files: ['bin', 'LICENSE', 'README.md', 'THIRD_PARTY_NOTICES.md'],
      optionalDependencies,
    }),
  )
  await chmod(join(metaDirectory, 'bin/akone.mjs'), 0o755)
  publishOrder.push(relative(ROOT, metaDirectory))
  await writeFile(join(npmOutput, 'publish-order.txt'), `${publishOrder.join('\n')}\n`)
  return { npmOutput, packages: publishOrder }
}

export function parseChecksums(source) {
  const checksums = new Map()
  for (const line of source.split(/\r?\n/)) {
    const match = line.trim().match(/^([0-9a-fA-F]{64})\s+\*?([^/]+)$/)
    if (match) checksums.set(match[2], match[1].toLowerCase())
  }
  return checksums
}

export async function stageHomebrew({ version, assets = DEFAULT_ASSETS, output = DEFAULT_OUTPUT }) {
  version = normalizeVersion(version)
  assertSafeOutput(output)
  const checksums = parseChecksums(await readFile(join(assets, 'checksums.txt'), 'utf8'))
  const intelName = `akone_${version}_darwin_amd64.tar.gz`
  const armName = `akone_${version}_darwin_arm64.tar.gz`
  const intelChecksum = checksums.get(intelName)
  const armChecksum = checksums.get(armName)
  if (!intelChecksum || !armChecksum) throw new Error('checksums.txt is missing a macOS akone archive')

  const formulaDirectory = join(output, 'homebrew-tap/Formula')
  await rm(join(output, 'homebrew-tap'), { force: true, recursive: true })
  await mkdir(formulaDirectory, { recursive: true })
  const base = `https://github.com/Payhon/AppKernia/releases/download/v${version}`
  const formula = `class Akone < Formula
  desc "AppKernia all-in-one server and agent-friendly CLI"
  homepage "https://github.com/Payhon/AppKernia"
  version "${version}"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "${base}/${armName}"
      sha256 "${armChecksum}"
    else
      url "${base}/${intelName}"
      sha256 "${intelChecksum}"
    end
  end

  def install
    bin.install "akone"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/akone version --json")
  end
end
`
  const formulaPath = join(formulaDirectory, 'akone.rb')
  await writeFile(formulaPath, formula)
  return { formulaPath }
}

function parseOptions(argv) {
  const command = argv[0]
  const options = { command, assets: DEFAULT_ASSETS, output: DEFAULT_OUTPUT }
  for (let index = 1; index < argv.length; index += 1) {
    const argument = argv[index]
    if (['--version', '--assets', '--output'].includes(argument)) {
      options[argument.slice(2)] = argv[index + 1]
      index += 1
    } else {
      throw new Error(`unknown argument: ${argument}`)
    }
  }
  options.version = normalizeVersion(options.version)
  options.assets = resolve(options.assets)
  options.output = resolve(options.output)
  return options
}

async function main() {
  const options = parseOptions(process.argv.slice(2))
  if (options.command === 'npm') {
    const result = await stageNpm(options)
    process.stdout.write(`release.npm.packages=${result.packages.length}\nrelease.npm.dir=${result.npmOutput}\n`)
  } else if (options.command === 'homebrew') {
    const result = await stageHomebrew(options)
    process.stdout.write(`release.homebrew.formula=${result.formulaPath}\n`)
  } else {
    throw new Error('first argument must be npm or homebrew')
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch((error) => {
    process.stderr.write(`release.stage.error=${error instanceof Error ? error.message : String(error)}\n`)
    process.exitCode = 1
  })
}
