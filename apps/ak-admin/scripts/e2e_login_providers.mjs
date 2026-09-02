import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const modulePath = process.env.AK_PLAYWRIGHT_MODULE;
if (!modulePath) throw new Error("AK_PLAYWRIGHT_MODULE is required");
const { chromium } = await import(modulePath);

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const base = (process.env.AK_E2E_BASE_URL ?? "http://127.0.0.1:4173").replace(/\/$/, "");
const chrome = process.env.AK_CHROME_PATH ?? "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const output = path.join(root, "output/playwright/login-providers");
const axePath = path.join(root, "apps/ak-admin/node_modules/axe-core/axe.min.js");
await mkdir(output, { recursive: true });

const tenantId = "523e4567-e89b-12d3-a456-426614174000";
const appId = "623e4567-e89b-12d3-a456-426614174000";
const userId = "423e4567-e89b-12d3-a456-426614174000";
const providerCodes = ["wechat", "github", "apple", "google"];
const permissions = [
  "app.application.read",
  "app.login_provider_binding.read",
  "app.login_provider_binding.update",
  "app.scanner_config.read",
  "app.scanner_config.update",
  "app.share_binding.read",
  "app.share_binding.update",
  "sys.login_provider_config.read",
  "sys.login_provider_config.create",
  "sys.login_provider_config.update",
  "sys.login_provider_config.delete",
  "sys.login_provider_config.rotate_secret",
  "sys.login_provider_config.preflight",
];

const application = {
  id: appId,
  tenant_id: tenantId,
  appid: "__UNI__APPKERNIA",
  appid_pending: false,
  app_type: "uni_app_x",
  code: "oauth-fixture",
  name: "AppKernia OAuth Fixture",
  description: "Controlled browser fixture for login-provider UI evidence.",
  introduction: "",
  remark: "API fixture; not a live provider acceptance environment.",
  status: "active",
  default_locale: "zh-CN",
  registration_enabled: true,
  registration_verification_mode: "email_otp",
  owner_type: "tenant",
  owner_id: tenantId,
  creator_user_id: userId,
  icon_file_id: null,
  managers: [userId],
  members: [],
  screenshots: [],
  channels: [],
  store_listings: [],
  is_default: true,
  lock_version: 3,
  startup: {
    translations: {
      "zh-CN": { display_name: "AppKernia", subtitle: "" },
      "en-US": { display_name: "AppKernia", subtitle: "" },
    },
    onboarding_enabled: false,
    draft_slides: [],
    published_version: 0,
    published_at: null,
    draft_changed: false,
  },
  created_at: "2026-09-01T02:00:00Z",
  updated_at: "2026-09-01T02:00:00Z",
};

const field = (name, location, valueType, required, secret, maxLength = 2048) => ({
  name,
  location,
  value_type: valueType,
  required,
  secret,
  max_length: maxLength,
  help_key: `login_providers.field_help.${name}`,
});

const catalog = [
  {
    provider_code: "wechat",
    display_name_key: "login_providers.provider.wechat",
    icon_key: "wechat",
    authorization_kind: "native_code",
    supported_platforms: ["android", "ios", "harmony"],
    build_variants: ["android_google", "android_china", "ios", "harmony"],
    config_schema_version: 1,
    requires_secret: true,
    fields: [
      field("external_client_id", "external_client_id", "string", true, false, 255),
      field("android_package_name", "public_config", "string", false, false, 255),
      field("android_app_signature", "public_config", "string", false, false, 256),
      field("ios_bundle_id", "public_config", "string", false, false, 255),
      field("ios_universal_link", "public_config", "url", false, false),
      field("harmony_bundle_name", "public_config", "string", false, false, 255),
      field("app_secret", "secret", "string", true, true, 4096),
    ],
    help_url: "https://open.weixin.qq.com/",
  },
  {
    provider_code: "github",
    display_name_key: "login_providers.provider.github",
    icon_key: "github",
    authorization_kind: "browser_ticket",
    supported_platforms: ["android", "ios", "harmony"],
    build_variants: ["android_google", "android_china", "ios", "harmony"],
    config_schema_version: 1,
    requires_secret: true,
    fields: [
      field("external_client_id", "external_client_id", "string", true, false, 255),
      field("app_return_uri", "public_config", "url", true, false),
      field("client_secret", "secret", "string", true, true, 4096),
    ],
    help_url: "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps",
  },
  {
    provider_code: "apple",
    display_name_key: "login_providers.provider.apple",
    icon_key: "apple",
    authorization_kind: "native_id_token",
    supported_platforms: ["ios"],
    build_variants: ["ios"],
    config_schema_version: 1,
    requires_secret: true,
    fields: [
      field("external_client_id", "external_client_id", "string", true, false, 255),
      field("team_id", "public_config", "string", true, false, 10),
      field("key_id", "public_config", "string", true, false, 10),
      field("private_key_p8", "secret", "pem", true, true, 65536),
    ],
    help_url: "https://developer.apple.com/sign-in-with-apple/get-started/",
  },
  {
    provider_code: "google",
    display_name_key: "login_providers.provider.google",
    icon_key: "google",
    authorization_kind: "native_id_token",
    supported_platforms: ["android"],
    build_variants: ["android_google"],
    config_schema_version: 1,
    requires_secret: false,
    fields: [
      field("external_client_id", "external_client_id", "string", true, false, 255),
      field("android_package_name", "public_config", "string", true, false, 255),
      field("android_certificate_sha256", "public_config", "string_array", true, false, 95),
    ],
    help_url: "https://developer.android.com/identity/sign-in/credential-manager-siwg",
  },
];

const ids = Object.fromEntries(providerCodes.map((code, index) => [code, `${String(index + 1).padStart(8, "0")}-1111-4111-8111-111111111111`]));
const configs = [
  {
    provider_code: "wechat",
    name: "WeChat Production",
    external_client_id: "wx0123456789abcdef",
    public_config: {
      android: { enabled: true, package_name: "com.appkernia.app", app_signature: "fixture-signature" },
      ios: { enabled: true, bundle_id: "com.appkernia.app", universal_link: "https://links.example.test/wechat/" },
      harmony: { enabled: true, bundle_name: "com.appkernia.app" },
    },
  },
  {
    provider_code: "github",
    name: "GitHub Production",
    external_client_id: "Iv1.fixtureclientid",
    public_config: { app_return_uri: "https://links.example.test/oauth/github" },
    callback_uri: "https://api.example.test/api/v1/auth/oauth/github/browser-callback",
  },
  {
    provider_code: "apple",
    name: "Apple Production",
    external_client_id: "com.appkernia.app",
    public_config: { team_id: "FIXTURE1234", key_id: "KEYID12345" },
  },
  {
    provider_code: "google",
    name: "Google Production",
    external_client_id: "fixture.apps.googleusercontent.com",
    public_config: {
      android_package_name: "com.appkernia.app",
      android_certificate_sha256: ["00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF"],
    },
  },
].map((config, index) => ({
  id: ids[config.provider_code],
  description: "Controlled non-secret API fixture.",
  config_schema_version: 1,
  secret_field_names: config.provider_code === "google" ? [] : [config.provider_code === "apple" ? "private_key_p8" : config.provider_code === "github" ? "client_secret" : "app_secret"],
  has_secret: config.provider_code !== "google",
  credential_fingerprint: `fixture-${config.provider_code}`,
  status: "active",
  last_preflight_at: "2026-09-01T03:00:00Z",
  last_preflight_status: "ready",
  last_preflight_issues: [],
  binding_count: index + 1,
  lock_version: 2,
  created_at: "2026-09-01T02:00:00Z",
  updated_at: "2026-09-01T03:00:00Z",
  ...config,
}));

const bindings = providerCodes.map((code, index) => ({
  id: `${String(index + 5).padStart(8, "0")}-2222-4222-8222-222222222222`,
  app_id: appId,
  provider_code: code,
  login_provider_config_id: ids[code],
  config_name: configs[index].name,
  config_status: "active",
  preflight_status: "ready",
  enabled: code === "wechat" || code === "github",
  sort_order: (index + 1) * 10,
  lock_version: 1,
  updated_at: "2026-09-01T03:00:00Z",
}));

function envelope(data) {
  return JSON.stringify({ code: "OK", message: "OK", data, request_id: "login-provider-fixture" });
}

function json(route, data, status = 200) {
  return route.fulfill({ status, contentType: "application/json", body: envelope(data) });
}

async function routeHandler(route) {
  const request = route.request();
  const url = new URL(request.url());
  const pathname = url.pathname;
  const method = request.method();
  if (pathname.endsWith("/auth/public-config")) return json(route, { locale: "zh-CN", default_locale: "zh-CN", supported_locales: ["zh-CN", "en-US"], feature_flags: {}, settings: {} });
  if (pathname.endsWith("/auth/csrf-token")) return json(route, { csrf_token: "fixture-csrf-token-long-enough" });
  if (pathname.endsWith("/auth/token/refresh")) return json(route, { access_token: "fixture-access-token", token_type: "Bearer", expires_in: 900, csrf_token: "fixture-csrf-token-long-enough" });
  if (pathname.endsWith("/auth/login")) return json(route, { access_token: "fixture-access-token", token_type: "Bearer", expires_in: 900, csrf_token: "fixture-csrf-token-long-enough" });
  if (pathname.endsWith("/auth/context")) return json(route, {
    user: { id: userId, email: "fixture@appkernia.test", display_name: "OAuth Fixture", locale: "zh-CN", time_zone: "UTC", avatar_url: null },
    active_tenant: { id: tenantId, name: "OAuth Fixture Tenant", code: "oauth-fixture" },
    available_tenants: [], roles: [], permissions, menus: [], feature_flags: {}, menu_revision: 1, permission_revision: 1, server_time: "2026-09-01T03:00:00Z",
  });
  if (pathname.endsWith("/me") && method === "PATCH") return json(route, { id: userId, email: "fixture@appkernia.test", display_name: "OAuth Fixture", locale: request.postDataJSON().locale, time_zone: "UTC", avatar_url: null });
  if (pathname.endsWith("/dictionaries/system.language")) {
    const english = (request.headers()["accept-language"] ?? "").startsWith("en-US");
    return json(route, { code: "system.language", locale: english ? "en-US" : "zh-CN", extension_policy: "fixed", items: [
      { value: "zh-CN", label: english ? "Simplified Chinese" : "简体中文", is_default: true, extra: {} },
      { value: "en-US", label: "English", is_default: false, extra: {} },
    ] });
  }
  const dashboardWindow = { range: url.searchParams.get("range") ?? "30d", start_at: "2026-08-03T00:00:00Z", end_at: "2026-09-01T00:00:00Z" };
  if (pathname.endsWith("/dashboard/summary")) return json(route, { ...dashboardWindow, metrics: [] });
  if (pathname.endsWith("/dashboard/trends")) return json(route, { ...dashboardWindow, series: [] });
  if (pathname.endsWith("/dashboard/activity")) return json(route, { ...dashboardWindow, operations: [], failed_jobs: [], security_events: [] });
  if (pathname.endsWith("/login-provider-catalog")) return json(route, { items: catalog });
  if (pathname.endsWith(`/apps/${appId}/login-provider-bindings`)) return json(route, { items: bindings });
  if (pathname.endsWith("/login-provider-configs")) return json(route, { items: configs, page: 1, page_size: Number(url.searchParams.get("page_size") ?? 20), total: configs.length });
  if (pathname.endsWith("/apps") && method === "GET") return json(route, { items: [application], total: 1 });
  if (pathname.endsWith(`/apps/${appId}/share-bindings`)) return json(route, []);
  if (pathname.endsWith("/share-configs")) return json(route, { items: [], page: 1, page_size: 100, total: 0 });
  if (pathname.endsWith(`/apps/${appId}/scanner-config`)) return json(route, { app_id: appId, enabled: false, trusted_hosts: [], lock_version: 0, updated_at: null });
  return json(route, {});
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function navigate(page, pathname) {
  await page.evaluate((value) => {
    window.history.pushState({}, "", value);
    window.dispatchEvent(new PopStateEvent("popstate"));
  }, pathname);
}

async function markFixture(page) {
  await page.evaluate(() => {
    if (document.querySelector("[data-ak-oauth-fixture]")) return;
    const marker = document.createElement("div");
    marker.dataset.akOauthFixture = "true";
    marker.setAttribute("aria-hidden", "true");
    marker.textContent = "API FIXTURE · NOT PLATFORM ACCEPTANCE";
    Object.assign(marker.style, {
      position: "fixed", top: "8px", right: "8px", zIndex: "99999", padding: "4px 8px",
      color: "#7a2e0b", background: "#fff7e6", border: "1px solid #fa8c16", borderRadius: "4px",
      font: "600 11px/1.4 system-ui, sans-serif", letterSpacing: ".02em", pointerEvents: "none",
    });
    document.body.append(marker);
  });
}

async function assertNoOverflow(page) {
  const size = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  assert(size.scrollWidth <= size.clientWidth, `page overflow: ${JSON.stringify(size)}`);
  return size;
}

async function audit(page, name, evidence) {
  await markFixture(page);
  const overflow = await assertNoOverflow(page);
  const result = await page.evaluate(async () => await globalThis.axe.run(
    { exclude: [[".ant-table-measure-row"]] },
    { resultTypes: ["violations"] },
  ));
  const severe = result.violations.filter((item) => item.impact === "critical" || item.impact === "serious");
  evidence.states[name] = {
    viewport: await page.evaluate(() => ({ width: innerWidth, height: innerHeight })),
    horizontal_overflow: overflow.scrollWidth > overflow.clientWidth,
    critical_or_serious: severe.length,
    violations: result.violations.map((item) => ({
      id: item.id,
      impact: item.impact,
      nodes: item.nodes.length,
      targets: item.nodes.map((node) => node.target),
    })),
  };
}

async function capture(page, name, evidence) {
  await page.waitForTimeout(600);
  await audit(page, name, evidence);
  await page.screenshot({ path: path.join(output, `${name}.png`), fullPage: name.startsWith("config.") });
}

async function setLocale(page, currentLabel, option, heading) {
  await page.getByLabel(currentLabel).click();
  await page.mouse.move(0, 0);
  await page.getByRole("menuitem", { name: option }).evaluate((element) => element.click());
  await page.getByRole("heading", { name: heading }).waitFor();
}

async function openGuide(page, locale) {
  await page.getByRole("button", { name: locale === "zh-CN" ? "查看第三方登录申请与接入指引" : "View provider application and integration guidance" }).click();
  const title = locale === "zh-CN" ? "第三方登录平台申请指引" : "Third-party Sign-in Provider Guide";
  const dialog = page.getByRole("dialog", { name: title });
  await dialog.waitFor();
  return dialog;
}

const guideAllowlist = new Set([
  "https://open.weixin.qq.com/",
  "https://doc.dcloud.net.cn/uni-app-x/api/sign-in.html",
  "https://github.com/settings/developers",
  "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps",
  "https://developer.apple.com/account/resources/authkeys/list",
  "https://developer.apple.com/sign-in-with-apple/get-started/",
  "https://console.cloud.google.com/apis/credentials",
  "https://developer.android.com/identity/sign-in/credential-manager-siwg",
]);

async function assertGuideLinks(dialog, locale) {
  const names = locale === "zh-CN" ? ["微信", "GitHub", "Apple", "Google"] : ["WeChat", "GitHub", "Apple", "Google"];
  const observed = new Set();
  for (const name of names) {
    await dialog.getByRole("tab", { name }).click();
    const links = await dialog.getByRole("tabpanel").getByRole("link").evaluateAll((elements) => elements.map((element) => element.href));
    assert(links.length === 2, `${name} guide link count: ${String(links.length)}`);
    for (const href of links) {
      assert(guideAllowlist.has(href), `guide link outside allowlist: ${href}`);
      observed.add(href);
    }
  }
  assert(observed.size === guideAllowlist.size, `guide allowlist coverage: ${String(observed.size)}`);
  await dialog.getByRole("tab", { name: names[0] }).click();
  return [...observed].sort();
}

async function openClientConfig(page, locale) {
  const menuLabel = locale === "zh-CN" ? "AppKernia OAuth Fixture 的操作" : "Actions for AppKernia OAuth Fixture";
  await page.getByRole("button", { name: menuLabel }).click();
  await page.getByRole("menuitem", { name: locale === "zh-CN" ? "客户端配置" : "Client configuration" }).click();
  const dialog = page.getByRole("dialog", { name: locale === "zh-CN" ? "客户端配置 · AppKernia OAuth Fixture" : "Client configuration · AppKernia OAuth Fixture" });
  await dialog.waitFor();
  await dialog.getByRole("tab", { name: locale === "zh-CN" ? "第三方登录" : "Third-party sign-in" }).click();
  await dialog.getByText(locale === "zh-CN" ? "按 App 开启第三方登录" : "Enable third-party sign-in per App").waitFor();
  return dialog;
}

const evidence = {
  fixture: {
    type: "controlled API route fixture",
    live_backend: false,
    live_provider_credentials: false,
    platform_authorization_acceptance: false,
    marker: "API FIXTURE · NOT PLATFORM ACCEPTANCE",
  },
  states: {},
  console_errors: [],
  page_errors: [],
};
const browser = await chromium.launch({ executablePath: chrome, headless: true });
evidence.browser = { version: browser.version(), executable: chrome };
const context = await browser.newContext({ locale: "zh-CN", viewport: { width: 1440, height: 900 }, colorScheme: "light" });
const page = await context.newPage();
page.setDefaultTimeout(20000);
await page.route("**/admin-api/**", routeHandler);
page.on("console", (message) => { if (message.type() === "error") evidence.console_errors.push(message.text()); });
page.on("pageerror", (error) => { evidence.page_errors.push(error.message); });

try {
  await page.goto(`${base}/login`, { waitUntil: "domcontentloaded" });
  if (await page.locator("#login-email").isVisible()) {
    await page.locator("#login-email").fill("fixture@appkernia.test");
    await page.locator("#login-password").fill("fixture-password");
    await page.locator('button[type="submit"]').click();
  }
  await page.waitForURL("**/dashboard**");
  await page.addScriptTag({ path: axePath });

  await navigate(page, "/system/settings/login-providers");
  await page.getByRole("heading", { name: "第三方登录配置" }).waitFor();
  await capture(page, "config.zh-CN.1440", evidence);
  let guide = await openGuide(page, "zh-CN");
  evidence.fixture.official_help_links = await assertGuideLinks(guide, "zh-CN");
  await capture(page, "help.zh-CN.1440", evidence);
  await page.keyboard.press("Escape");
  await guide.waitFor({ state: "hidden" });

  await page.setViewportSize({ width: 375, height: 812 });
  await capture(page, "config.zh-CN.375", evidence);
  guide = await openGuide(page, "zh-CN");
  await capture(page, "help.zh-CN.375", evidence);
  await page.keyboard.press("Escape");
  await guide.waitFor({ state: "hidden" });

  await page.setViewportSize({ width: 1440, height: 900 });
  await setLocale(page, "显示语言", "English", "Third-party Sign-in");
  await capture(page, "config.en-US.1440", evidence);
  guide = await openGuide(page, "en-US");
  await assertGuideLinks(guide, "en-US");
  await capture(page, "help.en-US.1440", evidence);
  await page.keyboard.press("Escape");
  await guide.waitFor({ state: "hidden" });

  await page.setViewportSize({ width: 375, height: 812 });
  await capture(page, "config.en-US.375", evidence);
  guide = await openGuide(page, "en-US");
  await capture(page, "help.en-US.375", evidence);
  await page.keyboard.press("Escape");
  await guide.waitFor({ state: "hidden" });

  await page.setViewportSize({ width: 1440, height: 900 });
  await navigate(page, "/app/applications");
  await page.getByRole("heading", { name: "Application management" }).waitFor();
  let clientConfig = await openClientConfig(page, "en-US");
  await capture(page, "client-config.en-US.1440", evidence);
  await page.keyboard.press("Escape");
  await clientConfig.waitFor({ state: "hidden" });

  await setLocale(page, "Display language", "简体中文", "应用管理");
  await page.setViewportSize({ width: 375, height: 812 });
  clientConfig = await openClientConfig(page, "zh-CN");
  await capture(page, "client-config.zh-CN.375", evidence);

  evidence.coverage = { locales: ["zh-CN", "en-US"], viewports: [375, 1440], help_providers: providerCodes, client_configuration_tab_order: ["share", "scanner", "login-providers"] };
  await writeFile(path.join(output, "e2e-results.json"), `${JSON.stringify(evidence, null, 2)}\n`, "utf8");
  assert(evidence.console_errors.length === 0, `unexpected console errors: ${JSON.stringify(evidence.console_errors)}`);
  assert(evidence.page_errors.length === 0, `unexpected page errors: ${JSON.stringify(evidence.page_errors)}`);
  const inaccessible = Object.entries(evidence.states).filter(([, state]) => state.critical_or_serious > 0);
  assert(inaccessible.length === 0, `axe serious/critical states: ${JSON.stringify(inaccessible)}`);
  console.log("login-provider-e2e:passed");
} finally {
  await context.close();
  await browser.close();
}
