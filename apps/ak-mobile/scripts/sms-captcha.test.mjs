import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { test } from 'node:test';

const ts = createRequire(new URL('../../ak-admin/package.json', import.meta.url))('typescript');
const root = new URL('../', import.meta.url);
const source = (path) => readFileSync(new URL(path, root), 'utf8');
const Point = class { constructor(x, y) { this.x = x; this.y = y; } };
const Response = class { constructor(type, points, point, angle) { Object.assign(this, { type, points, point, angle }); } };
const script = source('uni_modules/ak-interactive-captcha/components/ak-interactive-captcha/ak-interactive-captcha.uvue')
  .match(/<script lang="uts">([\s\S]*?)<\/script>/)[1].replace(/^import .*$/gm, '');
const code = ts.transpileModule(script, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
const exports = {};
const emitted = [];
const uni = {
  getSystemInfoSync: () => ({ windowWidth: 390 }),
  getElementById: () => ({ getBoundingClientRect: () => ({ left: 10, top: 20, width: 150, height: 110 }) }),
};
new Function('exports', 'akI18n', 'AkCaptchaPoint', 'AkInteractiveCaptchaResponse', 'uni', code)(
  exports, { t: (key) => key }, Point, Response, uni,
);
const options = exports.default;
const challenge = (type) => ({
  id: 'id', token: 'token', type, expiresInSeconds: 0,
  image: { width: 300, height: 220 }, requiredPoints: 2,
  tileImage: { width: 60, height: 60 }, initialPoint: new Point(10, 50), thumbImage: { width: 80, height: 80 },
});
const instance = (type) => {
  const vm = { open: true, loading: false, errorText: '', challenge: challenge(type), ...options.data(), $emit: (...args) => emitted.push(args) };
  for (const [name, method] of Object.entries(options.methods)) vm[name] = method.bind(vm);
  for (const [name, getter] of Object.entries(options.computed)) Object.defineProperty(vm, name, { get: () => getter.call(vm) });
  vm.reset();
  return vm;
};
const touch = (x, y) => ({ changedTouches: [{ clientX: x, clientY: y }], touches: [] });

test('click, slide, drag and rotate return normalized original-image answers', () => {
  const click = instance('click');
  click.clickStage(touch(85, 75));
  assert.deepEqual(click.points[0], new Point(150, 110));
  click.undo();
  assert.equal(click.points.length, 0);

  const slide = instance('slide');
  slide.slide({ detail: { value: 50 } });
  assert.deepEqual([slide.dragX, slide.dragY], [120, 50]);

  const drag = instance('drag');
  drag.dragStart(touch(30, 40));
  drag.dragMove(touch(45, 51));
  assert.deepEqual([drag.dragX, drag.dragY], [40, 72]);

  const rotate = instance('rotate');
  rotate.challenge.thumbImage.width = 100;
  assert.match(rotate.rotateStyle, /margin-left:-50px/);
  rotate.rotate({ detail: { value: 181.4 } });
  rotate.confirm();
  assert.equal(emitted.at(-1)[1].angle, 181);
});

test('challenge replacement, close and refresh clear prior answers and errors', () => {
  const vm = instance('click');
  vm.points.push(new Point(1, 2)); vm.angle = 90; vm.interacted = true; vm.localError = 'old';
  vm.challenge = challenge('drag'); vm.reset();
  assert.deepEqual([vm.points.length, vm.angle, vm.interacted, vm.dragX, vm.dragY, vm.localError], [0, 0, false, 10, 50, '']);
  vm.expire();
  assert.equal(vm.canConfirm, false);
  assert.equal(vm.localError, 'captcha.expired');
  vm.localError = 'old'; vm.close();
  assert.equal(vm.localError, '');
  vm.refresh();
  assert.deepEqual(emitted.slice(-2).map((item) => item[0]), ['close', 'refresh']);
});

test('every mobile SMS sender requests CAPTCHA first while email remains direct', () => {
  const files = [
    'components/ak-ui/ak-login-method-form/ak-login-method-form.uvue',
    'pages/auth/register/index.uvue',
    'pages/auth/forgot-password/index.uvue',
    'pages/profile/connections/index.uvue',
  ].map(source).join('\n');
  for (const scene of ['login', 'registration', 'password_reset', 'identifier_verify', 'step_up']) {
    assert.match(files, new RegExp(`scene:'${scene}'`));
  }
  assert.equal((files.match(/<ak-interactive-captcha/g) ?? []).length, 4);
  assert.match(source('components/ak-ui/ak-login-method-form/ak-login-method-form.uvue'), /identifierType\(\)==='mobile'.*openCaptcha\(\);return.*sendEmailOTP/s);
  assert.match(source('pages/auth/register/index.uvue'), /identifierType==='mobile'.*openCaptcha\(\);return.*sendRegistrationOTP\(this\.identifierType.*null/s);
  const repository = source('src/features/captcha/captcha-repository.uts');
  assert.match(repository, /MobileSMSCaptchaStepUpRequestWire=\{scene:'step_up',identifier_id:input\.identifierId/s);
  assert.match(repository, /MobileSMSCaptchaPublicRequestWire=\{scene:input\.scene,mobile:input\.mobile/s);
  assert.doesNotMatch(repository, /mobile:input\.mobile,identifier_id:input\.identifierId/);
});
