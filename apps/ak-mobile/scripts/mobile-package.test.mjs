import assert from 'node:assert/strict';
import test from 'node:test';

import {
  customAndroidConfig,
  customIosConfig,
  parseArguments,
  pathCandidates,
  redactedEnvironmentStatus,
} from './mobile-package.mjs';

test('parses a mode, target and dry-run flag', () => {
  assert.deepEqual(parseArguments(['custom-base', 'all', '--dry-run']), {
    mode: 'custom-base',
    target: 'all',
    dryRun: true,
  });
  assert.deepEqual(parseArguments(['release']), { invalid: true });
});

test('custom base configs always use AppKernia identity and custom mode', () => {
  const android = customAndroidConfig();
  const ios = customIosConfig('simulator');
  assert.equal(android.iscustom, true);
  assert.equal(android.android.packagename, 'com.appkernia.mobile');
  assert.equal(android.android.androidpacktype, 1);
  assert.equal(ios.iscustom, true);
  assert.equal(ios.ios.bundle, 'com.appkernia.mobile');
  assert.equal(ios.ios.channels, 'simulator');
});

test('Windows candidates support explicit and common executable paths', () => {
  const candidates = pathCandidates('hbuilder', 'win32', {
    HBUILDERX_CLI: 'D:\\HBuilderX\\cli.exe',
    ProgramFiles: 'C:\\Program Files',
    USERPROFILE: 'C:\\Users\\ak',
  });
  assert.equal(candidates[0], 'D:\\HBuilderX\\cli.exe');
  assert.ok(candidates.some((candidate) => candidate?.endsWith('HBuilderX\\cli.exe')));
});

test('signing status never returns secret values', () => {
  const previous = process.env.AK_ANDROID_CERT_PASSWORD;
  process.env.AK_ANDROID_CERT_PASSWORD = 'do-not-print-this';
  try {
    assert.deepEqual(redactedEnvironmentStatus(['AK_ANDROID_CERT_PASSWORD', 'AK_IOS_DIST_CERT_PASSWORD']), {
      AK_ANDROID_CERT_PASSWORD: '<configured>',
      AK_IOS_DIST_CERT_PASSWORD: '<missing>',
    });
  } finally {
    if (previous === undefined) delete process.env.AK_ANDROID_CERT_PASSWORD;
    else process.env.AK_ANDROID_CERT_PASSWORD = previous;
  }
});
