import assert from 'node:assert/strict'
import { chmod, mkdtemp, mkdir, readFile, writeFile } from 'node:fs/promises'
import { createHash } from 'node:crypto'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { after, test } from 'node:test'
import { rmSync } from 'node:fs'
import { spawnSync } from 'node:child_process'

import { isStableVersion, normalizeVersion, parseArgs } from './release.mjs'
import { stageAdmin } from './stage-admin.mjs'
import { npmPackageDefinitions, parseChecksums, stageHomebrew, stageNpm } from './stage-release-channels.mjs'
import {
  assertMatchingIntegrity,
  isMissingPackage,
  parsePackedIntegrity,
  parsePublishedIntegrity,
} from './verify-npm-package.mjs'

const temporaryDirectories = []
after(() => {
  for (const directory of temporaryDirectories) rmSync(directory, { force: true, recursive: true })
})

async function temporaryDirectory() {
  const directory = await mkdtemp(join(tmpdir(), 'akone-release-test-'))
  temporaryDirectories.push(directory)
  return directory
}

async function installerFixture(directory, archiveVersion, binaryVersion = archiveVersion) {
  const assets = join(directory, 'assets')
  const bundle = join(directory, 'bundle')
  const fakeBin = join(directory, 'bin')
  await mkdir(assets)
  await mkdir(bundle)
  await mkdir(fakeBin)
  await writeFile(
    join(bundle, 'akone'),
    `#!/bin/sh\n[ "\${1:-}" = version ] && [ "\${2:-}" = --json ] || exit 1\nprintf '%s\\n' '${JSON.stringify({ version: binaryVersion })}'\n`,
  )
  await chmod(join(bundle, 'akone'), 0o755)
  const releaseOs = process.platform === 'darwin' ? 'darwin' : 'linux'
  const releaseArch = process.arch === 'x64' ? 'amd64' : 'arm64'
  const archiveName = `akone_${archiveVersion}_${releaseOs}_${releaseArch}.tar.gz`
  const tar = spawnSync('tar', ['-czf', join(assets, archiveName), 'akone'], { cwd: bundle, encoding: 'utf8' })
  assert.equal(tar.status, 0, tar.stderr)
  const archive = await readFile(join(assets, archiveName))
  const checksum = createHash('sha256').update(archive).digest('hex')
  await writeFile(join(assets, 'checksums.txt'), `${checksum}  ${archiveName}\n`)
  await writeFile(
    join(fakeBin, 'curl'),
    '#!/bin/sh\nset -eu\nwhile [ "$#" -gt 0 ]; do\n  case "$1" in\n    --output) output="$2"; shift 2 ;;\n    https://*) url="$1"; shift ;;\n    *) shift ;;\n  esac\ndone\ncp "$AKONE_TEST_ASSETS/${url##*/}" "$output"\n',
  )
  await chmod(join(fakeBin, 'curl'), 0o755)
  return { assets, fakeBin }
}

function runInstaller({ assets, fakeBin, destination, version, environment = {} }) {
  return spawnSync('sh', ['scripts/install-akone.sh', '--version', version, '--install-dir', destination], {
    cwd: process.cwd(),
    encoding: 'utf8',
    env: {
      ...process.env,
      AKONE_TEST_ASSETS: assets,
      PATH: `${fakeBin}:${process.env.PATH}`,
      ...environment,
    },
  })
}

test('release versions are normalized and stable releases are distinguishable', () => {
  assert.equal(normalizeVersion('v1.2.3-preview.4'), '1.2.3-preview.4')
  assert.equal(isStableVersion('1.2.3'), true)
  assert.equal(isStableVersion('1.2.3-rc.1'), false)
  assert.throws(() => normalizeVersion('1.02.3'))
  assert.throws(() => normalizeVersion('1.2.3-preview.01'))
  assert.throws(() => normalizeVersion('1.2.3+build.1'))
  assert.throws(() => normalizeVersion('1.2'))
})

test('release stays dry unless publish is explicit', () => {
  assert.deepEqual(parseArgs(['--version', '1.2.3-preview.1'], {}), {
    publish: false,
    version: '1.2.3-preview.1',
  })
  assert.equal(parseArgs(['--version', '1.2.3-rc.1', '--publish'], {}).publish, true)
  assert.throws(() => parseArgs(['--version', '1.2.3-preview.1', '--publish', '--dry-run'], {}))
  assert.throws(() => parseArgs(['--version', '1.2.3-preview.1', '--unknown'], {}))
})

test('release workflows reject stable or unsigned tags', async () => {
  for (const workflow of ['.github/workflows/release.yml', '.github/workflows/release-channels.yml']) {
    const source = await readFile(workflow, 'utf8')
    assert.match(source, /stable releases are blocked until macOS notarization and Windows Authenticode signing are implemented/)
    assert.match(source, /\.verification\.verified/)
    assert.match(source, /merge-base --is-ancestor/)
    const externalActions = [...source.matchAll(/^\s*uses:\s*([^\s#]+)/gm)]
      .map((match) => match[1])
      .filter((reference) => !reference.startsWith('./'))
    assert.ok(externalActions.length > 0)
    for (const reference of externalActions) assert.match(reference, /^[^@\s]+@[0-9a-f]{40}$/)
  }
})

test('an existing npm package is skipped only when its packed integrity matches', async () => {
  const integrity = `sha512-${Buffer.alloc(64, 7).toString('base64')}`
  const different = `sha512-${Buffer.alloc(64, 8).toString('base64')}`
  assert.equal(parsePackedIntegrity(JSON.stringify([{ integrity }])), integrity)
  assert.equal(parsePublishedIntegrity(JSON.stringify(integrity)), integrity)
  assert.doesNotThrow(() => assertMatchingIntegrity('@appkernia/akone', '1.2.3-preview.1', integrity, integrity))
  assert.throws(
    () => assertMatchingIntegrity('@appkernia/akone', '1.2.3-preview.1', integrity, different),
    /different content/,
  )
  assert.equal(isMissingPackage({ status: 1, stdout: '', stderr: 'npm error code E404' }), true)
  assert.equal(isMissingPackage({ status: 1, stdout: '', stderr: 'npm error code E401' }), false)

  const workflow = await readFile('.github/workflows/release-channels.yml', 'utf8')
  assert.match(workflow, /verify-npm-package\.mjs/)
  assert.doesNotMatch(workflow, /already exists; skipping/)
})

test('Admin staging omits source maps and Vite metadata', async () => {
  const directory = await temporaryDirectory()
  const source = join(directory, 'source')
  const destination = join(directory, 'destination')
  await mkdir(join(source, 'assets'), { recursive: true })
  await mkdir(join(source, '.vite'), { recursive: true })
  await writeFile(join(source, 'index.html'), '<html></html>')
  await writeFile(join(source, 'assets/app.js'), 'ok')
  await writeFile(join(source, 'assets/app.js.map'), 'map')
  await writeFile(join(source, '.vite/manifest.json'), '{}')

  const result = await stageAdmin(source, destination)
  assert.equal(result.files, 2)
  assert.equal(await readFile(join(destination, 'assets/app.js'), 'utf8'), 'ok')
  await assert.rejects(readFile(join(destination, 'assets/app.js.map')))
  await assert.rejects(readFile(join(destination, '.vite/manifest.json')))
})

test('npm channel contains exactly five native platform packages', () => {
  const definitions = npmPackageDefinitions('1.2.3-preview.1')
  assert.equal(definitions.length, 5)
  assert.deepEqual(
    definitions.map((entry) => entry.packageName),
    [
      '@appkernia/akone-darwin-x64',
      '@appkernia/akone-darwin-arm64',
      '@appkernia/akone-linux-x64',
      '@appkernia/akone-linux-arm64',
      '@appkernia/akone-win32-x64',
    ],
  )
  assert.equal(definitions.at(-1).archiveName, 'akone_1.2.3-preview.1_windows_amd64.zip')
})

test('npm channel stages native archives and keeps the meta package last', async () => {
  const directory = await temporaryDirectory()
  const assets = join(directory, 'assets')
  const output = join(directory, 'output')
  await mkdir(assets)
  for (const definition of npmPackageDefinitions('1.2.3-preview.1')) {
    const bundle = join(directory, `${definition.goos}-${definition.goarch}`)
    const binary = definition.goos === 'windows' ? 'akone.exe' : 'akone'
    await mkdir(bundle)
    await writeFile(join(bundle, binary), `${definition.goos}/${definition.goarch}`)
    const command = definition.goos === 'windows' ? 'zip' : 'tar'
    const args = definition.goos === 'windows'
      ? ['-q', join(assets, definition.archiveName), binary]
      : ['-czf', join(assets, definition.archiveName), binary]
    const archive = spawnSync(command, args, { cwd: bundle, encoding: 'utf8' })
    assert.equal(archive.status, 0, archive.stderr)
  }

  const result = await stageNpm({ version: '1.2.3-preview.1', assets, output })
  assert.equal(result.packages.length, 6)
  assert.equal(result.packages.at(-1).endsWith('output/npm/akone'), true)
  const meta = JSON.parse(await readFile(join(result.npmOutput, 'akone/package.json'), 'utf8'))
  assert.equal(Object.keys(meta.optionalDependencies).length, 5)
  assert.equal(meta.optionalDependencies['@appkernia/akone-linux-x64'], '1.2.3-preview.1')
  assert.equal(await readFile(join(result.npmOutput, 'win32-x64/bin/akone.exe'), 'utf8'), 'windows/amd64')
})

test('Homebrew formula is generated from release checksums', async () => {
  const directory = await temporaryDirectory()
  const assets = join(directory, 'assets')
  const output = join(directory, 'output')
  await mkdir(assets)
  const intel = 'a'.repeat(64)
  const arm = 'b'.repeat(64)
  await writeFile(
    join(assets, 'checksums.txt'),
    `${intel}  akone_1.2.3_darwin_amd64.tar.gz\n${arm} *akone_1.2.3_darwin_arm64.tar.gz\n`,
  )
  assert.equal(parseChecksums(`${intel}  file\n`).get('file'), intel)
  const result = await stageHomebrew({ version: '1.2.3', assets, output })
  const formula = await readFile(result.formulaPath, 'utf8')
  assert.match(formula, /version "1\.2\.3"/)
  assert.match(formula, new RegExp(intel))
  assert.match(formula, new RegExp(arm))
  const ruby = spawnSync('ruby', ['-c', result.formulaPath], { encoding: 'utf8' })
  assert.equal(ruby.status, 0, ruby.stderr)
})

test('shell installer and npm wrapper parse', () => {
  const shell = spawnSync('sh', ['-n', 'scripts/install-akone.sh'], { encoding: 'utf8' })
  assert.equal(shell.status, 0, shell.stderr)
  const node = spawnSync(process.execPath, ['--check', 'release/npm/akone.mjs'], { encoding: 'utf8' })
  assert.equal(node.status, 0, node.stderr)
})

test('shell installer verifies and installs the matching archive', async () => {
  if (!['darwin', 'linux'].includes(process.platform) || !['arm64', 'x64'].includes(process.arch)) return
  const directory = await temporaryDirectory()
  const destination = join(directory, 'installed')
  const fixture = await installerFixture(directory, '1.2.3')
  const install = runInstaller({ ...fixture, destination, version: '1.2.3' })
  assert.equal(install.status, 0, install.stderr)
  assert.match(install.stdout, /akone\.install\.path=/)
  assert.equal((await readFile(join(destination, 'akone'), 'utf8')).startsWith('#!/bin/sh'), true)
})

test('shell installer rejects a mismatched archive or binary version', async () => {
  if (!['darwin', 'linux'].includes(process.platform) || !['arm64', 'x64'].includes(process.arch)) return

  const wrongArchiveDirectory = await temporaryDirectory()
  const wrongArchive = await installerFixture(wrongArchiveDirectory, '9.9.9')
  const archiveInstall = runInstaller({
    ...wrongArchive,
    destination: join(wrongArchiveDirectory, 'installed'),
    version: '1.2.3-preview.1',
  })
  assert.notEqual(archiveInstall.status, 0)
  assert.match(archiveInstall.stderr, /release has no artifact/)

  const wrongBinaryDirectory = await temporaryDirectory()
  const wrongBinary = await installerFixture(wrongBinaryDirectory, '1.2.3-preview.1', '9.9.9')
  const binaryInstall = runInstaller({
    ...wrongBinary,
    destination: join(wrongBinaryDirectory, 'installed'),
    version: '1.2.3-preview.1',
  })
  assert.notEqual(binaryInstall.status, 0)
  assert.match(binaryInstall.stderr, /downloaded akone version 9\.9\.9 does not match archive version 1\.2\.3-preview\.1/)
})

test('shell installer rejects an empty HOME and randomizes its same-directory target', async () => {
  if (!['darwin', 'linux'].includes(process.platform) || !['arm64', 'x64'].includes(process.arch)) return

  const emptyHomeEnvironment = { ...process.env }
  delete emptyHomeEnvironment.HOME
  delete emptyHomeEnvironment.AKONE_INSTALL_DIR
  const emptyHome = spawnSync('sh', ['scripts/install-akone.sh', '--version', '1.2.3-preview.1'], {
    cwd: process.cwd(),
    encoding: 'utf8',
    env: emptyHomeEnvironment,
  })
  assert.notEqual(emptyHome.status, 0)
  assert.match(emptyHome.stderr, /HOME is empty; pass --install-dir/)

  const directory = await temporaryDirectory()
  const destination = join(directory, 'installed')
  const mktempLog = join(directory, 'mktemp.log')
  const fixture = await installerFixture(directory, '1.2.3-preview.1')
  await writeFile(
    join(fixture.fakeBin, 'mktemp'),
    '#!/bin/sh\nprintf \'%s\\n\' "$1" >> "$AKONE_TEST_MKTEMP_LOG"\nexec /usr/bin/mktemp "$@"\n',
  )
  await chmod(join(fixture.fakeBin, 'mktemp'), 0o755)
  const installed = runInstaller({
    ...fixture,
    destination,
    version: '1.2.3-preview.1',
    environment: { AKONE_TEST_MKTEMP_LOG: mktempLog },
  })
  assert.equal(installed.status, 0, installed.stderr)
  const templates = (await readFile(mktempLog, 'utf8')).trim().split('\n')
  assert.equal(templates.at(-1), join(destination, '.akone.XXXXXX'))
  const installer = await readFile('scripts/install-akone.sh', 'utf8')
  assert.doesNotMatch(installer, /\.akone\.\$\$/)
})
