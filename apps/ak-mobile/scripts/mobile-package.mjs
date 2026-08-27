#!/usr/bin/env node

import { createHash } from 'node:crypto';
import {
  chmodSync,
  createWriteStream,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs';
import { homedir, tmpdir } from 'node:os';
import { basename, dirname, extname, join, resolve, win32 as winPath } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn, spawnSync } from 'node:child_process';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const mobileRoot = resolve(scriptDir, '..');
const repoRoot = resolve(mobileRoot, '../..');
const unpackageRoot = join(mobileRoot, 'unpackage');
const logRoot = join(unpackageRoot, 'mobile-package-logs');
const isWindows = process.platform === 'win32';
const manifest = JSON.parse(readFileSync(join(mobileRoot, 'manifest.json'), 'utf8'));
const nativeId = 'com.appkernia.mobile';
const iosBundleId = 'com.appkernia.mobile';

const errorPattern = /\[Error\]|编译失败|打包失败|错误：|程序已退出|安装鸿蒙工程依赖失败|ohpm ERROR|\bERROR\b/i;
function usage(exitCode = 0) {
  const output = `Usage:
  node apps/ak-mobile/scripts/mobile-package.mjs custom-base <preflight|assets|android|ios-simulator|ios-device|harmony|harmony-signed|all|verify> [--dry-run]
  node apps/ak-mobile/scripts/mobile-package.mjs release <preflight|android|ios|harmony-prepare|harmony|all|verify> [--dry-run]

Environment overrides:
  HBUILDERX_CLI, DEVECO_STUDIO_HOME, PYTHON_BIN
  AK_CUSTOM_IOS_TARGET
  AK_ANDROID_CERT_FILE, AK_ANDROID_CERT_ALIAS,
  AK_ANDROID_CERT_PASSWORD, AK_ANDROID_STORE_PASSWORD, AK_ANDROID_CHANNELS
  AK_IOS_DEV_PROFILE, AK_IOS_DEV_CERT_FILE, AK_IOS_DEV_CERT_PASSWORD
  AK_IOS_DIST_PROFILE, AK_IOS_DIST_CERT_FILE, AK_IOS_DIST_CERT_PASSWORD
`;
  (exitCode === 0 ? process.stdout : process.stderr).write(output);
  process.exit(exitCode);
}

function parseArguments(argv) {
  if (argv.includes('--help') || argv.includes('-h')) return { help: true };
  const dryRun = argv.includes('--dry-run');
  const positional = argv.filter((value) => value !== '--dry-run');
  if (positional.length !== 2) return { invalid: true };
  return { mode: positional[0], target: positional[1], dryRun };
}

function pathCandidates(kind, platform = process.platform, env = process.env) {
  const home = env.USERPROFILE || env.HOME || homedir();
  const pathJoin = platform === 'win32' ? winPath.join : join;
  if (kind === 'hbuilder') {
    if (platform === 'win32') {
      return [
        env.HBUILDERX_CLI,
        env.ProgramFiles && pathJoin(env.ProgramFiles, 'HBuilderX', 'cli.exe'),
        env['ProgramFiles(x86)'] && pathJoin(env['ProgramFiles(x86)'], 'HBuilderX', 'cli.exe'),
        env.LOCALAPPDATA && pathJoin(env.LOCALAPPDATA, 'Programs', 'HBuilderX', 'cli.exe'),
        pathJoin(home, 'HBuilderX', 'cli.exe'),
      ];
    }
    return [env.HBUILDERX_CLI, '/Applications/HBuilderX.app/Contents/MacOS/cli'];
  }
  if (kind === 'deveco') {
    if (platform === 'win32') {
      return [
        env.DEVECO_STUDIO_HOME,
        env.ProgramFiles && pathJoin(env.ProgramFiles, 'Huawei', 'DevEco Studio'),
        env.LOCALAPPDATA && pathJoin(env.LOCALAPPDATA, 'Programs', 'Huawei', 'DevEco Studio'),
      ];
    }
    return [env.DEVECO_STUDIO_HOME, '/Applications/DevEco-Studio.app/Contents'];
  }
  return [];
}

function commandOnPath(names) {
  for (const name of names) {
    const result = spawnSync(isWindows ? 'where.exe' : 'sh', isWindows ? [name] : ['-lc', `command -v ${name}`], {
      encoding: 'utf8',
      windowsHide: true,
    });
    if (result.status === 0) {
      const candidate = result.stdout.split(/\r?\n/).find(Boolean)?.trim();
      if (candidate) return candidate;
    }
  }
  return undefined;
}

function firstExisting(candidates) {
  return candidates.filter(Boolean).find((candidate) => existsSync(candidate));
}

function resolveTools({ needsHarmony = false, dryRun = false } = {}) {
  const hbuilder = firstExisting(pathCandidates('hbuilder')) || commandOnPath(['cli.exe', 'cli']);
  const pythonOverride = process.env.PYTHON_BIN;
  let python;
  let pythonPrefix = [];
  if (pythonOverride) {
    python = pythonOverride;
  } else if (isWindows && commandOnPath(['py.exe', 'py'])) {
    python = commandOnPath(['py.exe', 'py']);
    pythonPrefix = ['-3'];
  } else {
    python = commandOnPath(['python3', 'python']);
  }

  let deveco;
  let ohpm;
  let hvigor;
  if (needsHarmony) {
    deveco = firstExisting(pathCandidates('deveco'));
    if (deveco) {
      ohpm = join(deveco, 'tools', 'ohpm', 'bin', isWindows ? 'ohpm.bat' : 'ohpm');
      hvigor = join(deveco, 'tools', 'hvigor', 'bin', isWindows ? 'hvigorw.bat' : 'hvigorw');
    }
  }

  const missing = [];
  if (!hbuilder) missing.push('HBuilderX CLI (set HBUILDERX_CLI)');
  if (!python) missing.push('Python 3 (set PYTHON_BIN)');
  if (needsHarmony && (!deveco || !existsSync(ohpm) || !existsSync(hvigor))) {
    missing.push('DevEco Studio OHPM/Hvigor (set DEVECO_STUDIO_HOME)');
  }
  if (missing.length && !dryRun) throw new Error(`Missing toolchain: ${missing.join('; ')}`);
  return { hbuilder, python, pythonPrefix, deveco, ohpm, hvigor, missing };
}

function assertFiles(values, { dryRun, label }) {
  const missing = values.filter((value) => !value || !existsSync(value));
  if (missing.length && !dryRun) throw new Error(`${label}: missing ${missing.join(', ')}`);
  return missing;
}

function assertEnvironment(names, { dryRun, label }) {
  const missing = names.filter((name) => !process.env[name]);
  if (missing.length && !dryRun) throw new Error(`${label}: missing ${missing.join(', ')}`);
  return missing;
}

function redactedEnvironmentStatus(names) {
  return Object.fromEntries(names.map((name) => [name, process.env[name] ? '<configured>' : '<missing>']));
}

function printableCommand(command, args) {
  const quote = (value) => (/\s/.test(value) ? JSON.stringify(value) : value);
  return [command || '<missing-tool>', ...args].map(quote).join(' ');
}

function spawnOptions(command, options) {
  return {
    cwd: options.cwd || repoRoot,
    env: options.env || process.env,
    windowsHide: true,
    shell: isWindows && /\.(?:bat|cmd)$/i.test(command || ''),
  };
}

async function runCommand(label, command, args, options = {}) {
  process.stdout.write(`\n[${label}] ${printableCommand(command, args)}\n`);
  if (options.dryRun) return { code: 0, output: '' };
  if (!command) throw new Error(`${label}: executable is unavailable`);
  mkdirSync(logRoot, { recursive: true });
  const logPath = options.logName ? join(logRoot, options.logName) : undefined;
  const stream = logPath ? createWriteStream(logPath, { flags: 'w', mode: 0o600 }) : undefined;
  const child = spawn(command, args, { ...spawnOptions(command, options), stdio: ['ignore', 'pipe', 'pipe'] });
  let output = '';
  const consume = (chunk, destination) => {
    const text = chunk.toString();
    output += text;
    destination.write(chunk);
    stream?.write(chunk);
  };
  child.stdout.on('data', (chunk) => consume(chunk, process.stdout));
  child.stderr.on('data', (chunk) => consume(chunk, process.stderr));
  const code = await new Promise((resolvePromise, reject) => {
    child.on('error', reject);
    child.on('close', resolvePromise);
  });
  stream?.end();
  if (!options.allowNonZero && code !== 0) throw new Error(`${label}: exit ${code}`);
  if (!options.allowErrorMarkers && errorPattern.test(output)) throw new Error(`${label}: error marker found`);
  return { code, output, logPath };
}

function withoutProxy() {
  const env = { ...process.env };
  for (const key of ['HTTP_PROXY', 'HTTPS_PROXY', 'ALL_PROXY', 'http_proxy', 'https_proxy', 'all_proxy']) {
    delete env[key];
  }
  return env;
}

async function runPython(tools, args, options = {}) {
  return runCommand(options.label || basename(args[0]), tools.python, [...tools.pythonPrefix, ...args], options);
}

async function generateAssets(tools, dryRun) {
  await runPython(tools, [join(scriptDir, 'generate-native-icons.py')], {
    label: 'generate native assets',
    dryRun,
  });
  await runPython(tools, [join(scriptDir, 'verify-custom-base.py')], {
    label: 'verify native asset contract',
    dryRun,
  });
}

function writeSecurePackConfig(data) {
  const directory = mkdtempSync(join(tmpdir(), 'appkernia-hbuilder-pack-'));
  const configPath = join(directory, 'configure.json');
  writeFileSync(configPath, `${JSON.stringify({ data }, null, 2)}\n`, { mode: 0o600 });
  chmodSync(configPath, 0o600);
  return { directory, configPath };
}

async function withPackConfig(data, callback, dryRun) {
  if (dryRun) return callback('<secure-temporary-config>');
  const temporary = writeSecurePackConfig(data);
  try {
    return await callback(temporary.configPath);
  } finally {
    rmSync(temporary.directory, { recursive: true, force: true });
  }
}

async function openProject(tools, dryRun) {
  await runCommand('open HBuilderX project', tools.hbuilder, ['project', 'open', '--path', mobileRoot], {
    dryRun,
    allowErrorMarkers: true,
  });
}

async function closeProject(tools, dryRun) {
  try {
    await runCommand('close HBuilderX project', tools.hbuilder, ['project', 'close', '--path', mobileRoot], {
      dryRun,
      allowNonZero: true,
      allowErrorMarkers: true,
    });
  } catch {
    // Closing is best-effort; the build result is already known.
  }
}

async function cloudPack(tools, label, data, logName, dryRun) {
  await withPackConfig(
    data,
    (configPath) =>
      runCommand(label, tools.hbuilder, ['pack', '--config', configPath, '--project', mobileRoot], {
        dryRun,
        logName,
      }),
    dryRun,
  );
}

function customAndroidConfig() {
  return {
    platform: 'android',
    iscustom: true,
    sourceMap: false,
    ignoreWarnings: true,
    android: { packagename: nativeId, androidpacktype: 1 },
  };
}

function customIosConfig(target) {
  const data = {
    platform: 'ios',
    iscustom: true,
    sourceMap: false,
    ignoreWarnings: true,
    ios: { bundle: iosBundleId, supporteddevice: 'iPhone', channels: target === 'simulator' ? 'simulator' : 'phone' },
  };
  if (target === 'device') {
    data.ios.profile = process.env.AK_IOS_DEV_PROFILE;
    data.ios.certfile = process.env.AK_IOS_DEV_CERT_FILE;
    data.ios.certpassword = process.env.AK_IOS_DEV_CERT_PASSWORD;
  }
  return data;
}

function releaseAndroidConfig() {
  return {
    platform: 'android',
    iscustom: false,
    sourceMap: false,
    ignoreWarnings: false,
    android: {
      packagename: nativeId,
      androidpacktype: 0,
      certalias: process.env.AK_ANDROID_CERT_ALIAS,
      certfile: process.env.AK_ANDROID_CERT_FILE,
      certpassword: process.env.AK_ANDROID_CERT_PASSWORD,
      storepassword: process.env.AK_ANDROID_STORE_PASSWORD,
      ...(process.env.AK_ANDROID_CHANNELS ? { channels: process.env.AK_ANDROID_CHANNELS } : {}),
    },
  };
}

function releaseIosConfig() {
  return {
    platform: 'ios',
    iscustom: false,
    sourceMap: false,
    dsyms: true,
    ignoreWarnings: false,
    ios: {
      bundle: iosBundleId,
      supporteddevice: process.env.AK_IOS_SUPPORTED_DEVICE || 'iPhone',
      channels: 'phone',
      profile: process.env.AK_IOS_DIST_PROFILE,
      certfile: process.env.AK_IOS_DIST_CERT_FILE,
      certpassword: process.env.AK_IOS_DIST_CERT_PASSWORD,
    },
  };
}

function preflightIos(target, dryRun) {
  if (target === 'simulator') {
    if (process.platform !== 'darwin' && !dryRun) throw new Error('iOS simulator custom base requires macOS');
    return [];
  }
  const envNames = ['AK_IOS_DEV_PROFILE', 'AK_IOS_DEV_CERT_FILE', 'AK_IOS_DEV_CERT_PASSWORD'];
  const missingEnv = assertEnvironment(envNames, { dryRun, label: 'iOS custom device signing' });
  const missingFiles = assertFiles(
    [process.env.AK_IOS_DEV_PROFILE, process.env.AK_IOS_DEV_CERT_FILE].filter(Boolean),
    { dryRun, label: 'iOS custom device signing' },
  );
  return [...missingEnv, ...missingFiles];
}

function preflightReleasePlatform(platform, dryRun) {
  if (platform === 'android') {
    const names = [
      'AK_ANDROID_CERT_FILE',
      'AK_ANDROID_CERT_ALIAS',
      'AK_ANDROID_CERT_PASSWORD',
      'AK_ANDROID_STORE_PASSWORD',
    ];
    const missing = assertEnvironment(names, { dryRun, label: 'Android release signing' });
    assertFiles([process.env.AK_ANDROID_CERT_FILE].filter(Boolean), { dryRun, label: 'Android release signing' });
    return missing;
  }
  if (platform === 'ios') {
    const names = ['AK_IOS_DIST_PROFILE', 'AK_IOS_DIST_CERT_FILE', 'AK_IOS_DIST_CERT_PASSWORD'];
    const missing = assertEnvironment(names, { dryRun, label: 'iOS distribution signing' });
    assertFiles(
      [process.env.AK_IOS_DIST_PROFILE, process.env.AK_IOS_DIST_CERT_FILE].filter(Boolean),
      { dryRun, label: 'iOS distribution signing' },
    );
    return missing;
  }
  return [];
}

async function generateHarmonyProject(tools, dryRun) {
  const result = await runCommand(
    'generate Harmony native project',
    tools.hbuilder,
    ['pack', 'app-harmony', '--project', mobileRoot],
    {
      dryRun,
      env: withoutProxy(),
      logName: 'harmony-hbuilder.log',
      allowNonZero: true,
      allowErrorMarkers: true,
    },
  );
  if (!dryRun) {
    const nativeProfile = join(unpackageRoot, 'dist', 'dev', 'app-harmony', 'build-profile.json5');
    if (!/项目 .* 编译成功/.test(result.output) || !existsSync(nativeProfile)) {
      throw new Error('HBuilderX did not generate a compiled Harmony native project');
    }
  }
}

async function prepareHarmony(tools, mode, dryRun) {
  await runPython(tools, [join(scriptDir, 'prepare-harmony-native.py'), mode], {
    label: `prepare Harmony native project (${mode})`,
    dryRun,
  });
}

async function buildHarmonyNative(tools, { signing, buildMode, packageType, dryRun }) {
  const nativeRoot = join(unpackageRoot, 'dist', 'dev', 'app-harmony');
  await prepareHarmony(tools, signing === 'unsigned' ? '--unsigned' : '--require-signing', dryRun);
  const env = withoutProxy();
  env.DEVECO_SDK_HOME = join(tools.deveco || '<DEVECO_STUDIO_HOME>', 'sdk');
  await runCommand('install Harmony dependencies', tools.ohpm, ['install', '--all'], {
    cwd: nativeRoot,
    env,
    dryRun,
    logName: 'harmony-ohpm.log',
  });
  const isApp = packageType === 'app';
  const args = [
    '--mode',
    isApp ? 'project' : 'module',
    '-p',
    'product=default',
    ...(isApp ? [] : ['-p', 'module=entry@default']),
    '-p',
    `buildMode=${buildMode}`,
    isApp ? 'assembleApp' : 'assembleHap',
    '--no-daemon',
  ];
  await runCommand(`build Harmony ${buildMode} ${packageType.toUpperCase()}`, tools.hvigor, args, {
    cwd: nativeRoot,
    env,
    dryRun,
    logName: `harmony-${buildMode}-${packageType}.log`,
  });
}

function walkFiles(root, result = []) {
  if (!existsSync(root)) return result;
  for (const entry of readdirSync(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) walkFiles(path, result);
    else if (entry.isFile()) result.push(path);
  }
  return result;
}

function sha256(path) {
  return createHash('sha256').update(readFileSync(path)).digest('hex');
}

function verifyReleaseOutputs() {
  const files = walkFiles(unpackageRoot);
  const candidates = files.filter((path) => !path.includes(`${join('unpackage', 'debug')}`));
  const android = candidates.filter((path) => ['.apk', '.aab'].includes(extname(path).toLowerCase()));
  const ios = candidates.filter((path) => extname(path).toLowerCase() === '.ipa');
  const harmony = candidates.filter(
    (path) => extname(path).toLowerCase() === '.app' && !basename(path).toLowerCase().includes('unsigned'),
  );
  const missing = [];
  if (!android.length) missing.push('Android release APK/AAB');
  if (!ios.length) missing.push('iOS release IPA');
  if (!harmony.length) missing.push('Harmony release APP');
  if (missing.length) throw new Error(`Release artifacts not found: ${missing.join(', ')}`);
  for (const [platform, paths] of Object.entries({ android, ios, harmony })) {
    const newest = paths.sort((left, right) => statSync(right).mtimeMs - statSync(left).mtimeMs)[0];
    process.stdout.write(`${platform}: ${newest}\nsha256: ${sha256(newest)}\n`);
  }
}

async function verifyCustomBase(tools, dryRun) {
  await runPython(tools, [join(scriptDir, 'verify-custom-base.py'), '--artifacts'], {
    label: 'verify custom-base artifacts',
    dryRun,
  });
}

async function verifyCustomBaseSources(tools, dryRun) {
  await runPython(tools, [join(scriptDir, 'verify-custom-base.py')], {
    label: 'verify custom-base source contract',
    dryRun,
  });
}

function printDryRunSigningSummary(mode) {
  if (mode === 'release') {
    const names = [
      'AK_ANDROID_CERT_FILE',
      'AK_ANDROID_CERT_ALIAS',
      'AK_ANDROID_CERT_PASSWORD',
      'AK_ANDROID_STORE_PASSWORD',
      'AK_IOS_DIST_PROFILE',
      'AK_IOS_DIST_CERT_FILE',
      'AK_IOS_DIST_CERT_PASSWORD',
    ];
    process.stdout.write(`Signing inputs: ${JSON.stringify(redactedEnvironmentStatus(names))}\n`);
  }
}

async function customBase(target, dryRun) {
  const iosTarget = process.env.AK_CUSTOM_IOS_TARGET || (process.platform === 'darwin' ? 'simulator' : 'device');
  const needsHarmony = ['preflight', 'harmony', 'harmony-signed', 'all'].includes(target);
  const tools = resolveTools({ needsHarmony, dryRun });
  if (['preflight', 'all'].includes(target)) preflightIos(iosTarget, dryRun);
  if (target === 'ios-device') preflightIos('device', dryRun);
  if (target === 'ios-simulator') preflightIos('simulator', dryRun);
  if (target === 'preflight') {
    await verifyCustomBaseSources(tools, dryRun);
    process.stdout.write(`Custom-base preflight: PASS (iOS target: ${iosTarget})\n`);
    return;
  }
  if (target === 'assets') return generateAssets(tools, dryRun);
  if (target === 'verify') return verifyCustomBase(tools, dryRun);
  const allowed = new Set(['android', 'ios-simulator', 'ios-device', 'harmony', 'harmony-signed', 'all']);
  if (!allowed.has(target)) usage(2);
  await generateAssets(tools, dryRun);
  await openProject(tools, dryRun);
  try {
    if (target === 'android' || target === 'all') {
      await cloudPack(tools, 'build Android custom base', customAndroidConfig(), 'custom-base-android.log', dryRun);
    }
    if (target === 'ios-simulator' || (target === 'all' && iosTarget === 'simulator')) {
      await cloudPack(tools, 'build iOS simulator custom base', customIosConfig('simulator'), 'custom-base-ios-simulator.log', dryRun);
    }
    if (target === 'ios-device' || (target === 'all' && iosTarget === 'device')) {
      await cloudPack(tools, 'build iOS device custom base', customIosConfig('device'), 'custom-base-ios-device.log', dryRun);
    }
    if (target === 'harmony' || target === 'all') {
      await generateHarmonyProject(tools, dryRun);
      await buildHarmonyNative(tools, { signing: 'unsigned', buildMode: 'debug', packageType: 'hap', dryRun });
    }
    if (target === 'harmony-signed') {
      await buildHarmonyNative(tools, { signing: 'signed', buildMode: 'debug', packageType: 'hap', dryRun });
    }
  } finally {
    await closeProject(tools, dryRun);
  }
}

async function release(target, dryRun) {
  const needsHarmony = ['preflight', 'harmony-prepare', 'harmony', 'all'].includes(target);
  const tools = resolveTools({ needsHarmony, dryRun });
  printDryRunSigningSummary('release');
  if (['preflight', 'android', 'all'].includes(target)) preflightReleasePlatform('android', dryRun);
  if (['preflight', 'ios', 'all'].includes(target)) preflightReleasePlatform('ios', dryRun);
  if (target === 'preflight') {
    if (!dryRun) {
      await runPython(tools, [join(scriptDir, 'prepare-harmony-native.py'), '--check-signing'], {
        label: 'verify Harmony release signing',
      });
    }
    process.stdout.write('Release preflight: PASS\n');
    return;
  }
  if (target === 'verify') {
    if (dryRun) {
      process.stdout.write('Would verify newest Android APK/AAB, iOS IPA and signed Harmony APP hashes.\n');
      return;
    }
    verifyReleaseOutputs();
    return;
  }
  const allowed = new Set(['android', 'ios', 'harmony-prepare', 'harmony', 'all']);
  if (!allowed.has(target)) usage(2);
  await generateAssets(tools, dryRun);
  await openProject(tools, dryRun);
  try {
    if (target === 'android' || target === 'all') {
      await cloudPack(tools, 'build Android release', releaseAndroidConfig(), 'release-android.log', dryRun);
    }
    if (target === 'ios' || target === 'all') {
      await cloudPack(tools, 'build iOS release', releaseIosConfig(), 'release-ios.log', dryRun);
    }
    if (target === 'harmony-prepare') {
      await generateHarmonyProject(tools, dryRun);
      await prepareHarmony(tools, '--unsigned', dryRun);
      process.stdout.write('Harmony release project prepared; configure AppKernia release signing in DevEco Studio next.\n');
    }
    if (target === 'harmony' || target === 'all') {
      await buildHarmonyNative(tools, { signing: 'signed', buildMode: 'release', packageType: 'app', dryRun });
    }
  } finally {
    await closeProject(tools, dryRun);
  }
}

export {
  customAndroidConfig,
  customIosConfig,
  parseArguments,
  pathCandidates,
  redactedEnvironmentStatus,
  releaseAndroidConfig,
  releaseIosConfig,
};

const isMainModule = process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url);
if (isMainModule) {
  const parsed = parseArguments(process.argv.slice(2));
  if (parsed.help) usage(0);
  if (parsed.invalid || !['custom-base', 'release'].includes(parsed.mode)) usage(2);

  try {
    if (parsed.mode === 'custom-base') await customBase(parsed.target, parsed.dryRun);
    else await release(parsed.target, parsed.dryRun);
  } catch (error) {
    process.stderr.write(`mobile-package: ${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  }
}
