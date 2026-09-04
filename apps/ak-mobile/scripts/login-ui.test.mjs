import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { test } from 'node:test';

const ts = createRequire(new URL('../../ak-admin/package.json', import.meta.url))('typescript');
const root = new URL('../', import.meta.url);
const source = (path) => readFileSync(new URL(path, root), 'utf8');
const component = (path, stubs = {}) => {
  const script = source(path).match(/<script lang="uts">([\s\S]*?)<\/script>/)[1].replace(/^import .*$/gm, '');
  const code = ts.transpileModule(script, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  new Function('exports', ...Object.keys(stubs), code)(exports, ...Object.values(stubs));
  return exports.default;
};
const i18n = { t: (key) => key };
const instantiate = (options, props = {}) => {
  const vm = { ...props, ...options.data?.() };
  for (const [key, value] of Object.entries(options.methods ?? {})) vm[key] = value.bind(vm);
  for (const [key, value] of Object.entries(options.computed ?? {})) Object.defineProperty(vm, key, { get: () => value.call(vm) });
  return vm;
};
const config = (otp = false) => ({ registrationEnabled: () => true, otpEnabled: () => otp, emailOTPEnabled: () => otp, smsOTPEnabled: () => otp });
const switchPath = 'components/ak-ui/ak-login-method-switch/ak-login-method-switch.uvue';
const registerPath = 'pages/auth/register/index.uvue';

test('registration legal names are inline links with localized prose and unchanged destinations', () => {
  const markup = source(registerPath).match(/<view class="register-consent">([\s\S]*?)<\/view>/)[1];
  assert.doesNotMatch(markup, /<ak-button/);
  assert.equal((markup.match(/role="link"/g) ?? []).length, 2);
  for (const handler of ['privacy', 'terms']) assert.match(markup, new RegExp(`role="link" @click="${handler}"`));
  const keys = [...markup.matchAll(/t\('([^']+)'/g)].map((match) => match[1]);
  for (const locale of ['zh-CN', 'en-US']) {
    const catalog = JSON.parse(source(`locale/${locale}.json`));
    assert.equal(keys.map((key) => catalog[key]).join(''), locale === 'zh-CN'
      ? '请先阅读并同意隐私政策和用户协议'
      : 'Please review and agree to the Privacy policy and Terms of service');
  }
  const destinations = [];
  const vm = instantiate(component(registerPath, { akI18n: i18n, appPublicConfigRuntime: config(), uni: { navigateTo: ({ url }) => destinations.push(url) } }));
  vm.privacy(); vm.terms();
  assert.deepEqual(destinations, ['/pages/legal/privacy/index', '/pages/legal/terms/index']);
});

test('method selector is hidden for password-only and retained for two methods', () => {
  const options = component(switchPath, { akI18n: i18n });
  assert.equal(instantiate(options, { otpEnabled: false }).items.length, 1);
  assert.equal(instantiate(options, { otpEnabled: true }).items.length, 2);
  assert.match(source(switchPath), /<view v-if="items.length > 1" class="method-switch"/);
});

test('compact sheet stays below safe area on small, Dynamic Island and reduced-height windows', () => {
  for (const [height, top, bottom] of [[667, 20, 0], [874, 62, 34], [932, 62, 34], [300, 62, 0]]) {
    const options = component('components/ak-ui/ak-bottom-sheet/ak-bottom-sheet.uvue', {
      akI18n: i18n, uni: { getWindowInfo: () => ({ windowHeight: height, safeAreaInsets: { top, bottom } }) },
    });
    const vm = instantiate(options, { expanded: true, heightRatio: 0.75 });
    const actual = Number(vm.sheetStyle.match(/^height:([\d.]+)px/)[1]);
    assert.ok(actual <= height * 0.75);
    assert.ok(height - actual >= top + 12);
    assert.ok(Number(vm.sheetStyle.match(/min-height:([\d.]+)px/)[1]) <= actual);
  }
  assert.match(source('components/ak-ui/ak-auth-prompt/ak-auth-prompt.uvue'), /:height-ratio="0.75"/);
});

test('password login validates email regardless of disabled or previously selected SMS OTP', () => {
  const options = component('components/ak-ui/ak-login-method-form/ak-login-method-form.uvue', { akI18n: i18n, appPublicConfigRuntime: config() });
  const vm = instantiate(options);
  vm.email = ' person@example.invalid ';
  vm.otpType = 'mobile';
  assert.equal(vm.identifierType(), 'email');
  assert.equal(vm.identifierValue(), 'person@example.invalid');
  assert.equal(vm.validateIdentifier(), true);
  vm.method = 'otp';
  assert.equal(vm.validateIdentifier(), false);
});

test('registration stays visible and blocks submission until both legal documents load', () => {
  const pending = [];
  const options = component(registerPath, {
    akI18n: i18n, appPublicConfigRuntime: config(),
    legalRuntime: { getRepository: () => ({ publicDocument: (...args) => pending.push(args) }) },
    authRuntime: { getService: () => { throw new Error('must not submit before legal readiness'); } },
  });
  const vm = instantiate(options);
  vm.displayName = 'Test'; vm.email = 'person@example.invalid';
  assert.equal(vm.validBase(), true);
  vm.loadLegal();
  assert.equal(pending[0][0], 'privacy-policy');
  vm.submit(); vm.sendCode();
  pending[0][2]({ code: 'CONTENT.PAGE.NOT_FOUND', kind: 'unknown' });
  assert.equal(vm.configMessageKey, 'auth.register.legalUnavailable');
  assert.equal(vm.configState, 'error');
  vm.submit(); vm.sendCode();
  vm.loadLegal();
  pending[1][1]();
  assert.equal(pending[2][0], 'terms-of-service');
  assert.equal(vm.configState, 'loading');
  pending[2][1]();
  assert.equal(vm.configState, 'content');
  assert.equal(vm.email, 'person@example.invalid');
  assert.match(source(registerPath), /<ak-card class="form-card">/);
  assert.match(source(registerPath), /:disabled="configState!=='content'"/);
});

test('registration distinguishes disabled and network failure without allowing submission', () => {
  const options = component(registerPath, { akI18n: i18n, appPublicConfigRuntime: { ...config(), registrationEnabled: () => false } });
  const vm = instantiate(options);
  vm.loadLegal();
  assert.equal(vm.configState, 'disabled');
  assert.equal(vm.configMessageKey, 'auth.register.disabled');
  vm.legalFailure({ kind: 'network_unavailable', code: 'request.network_unavailable', messageKey: 'errors.request.network_unavailable' });
  assert.equal(vm.configState, 'offline');
  assert.equal(vm.configMessageKey, 'errors.request.network_unavailable');
});
