import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const modulePath = process.env.AK_PLAYWRIGHT_MODULE;
if (!modulePath) throw new Error("AK_PLAYWRIGHT_MODULE is required");
const { chromium } = await import(modulePath);

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const base = (process.env.AK_E2E_BASE_URL ?? "http://127.0.0.1:4173").replace(/\/$/, "");
const output = path.join(root, "output/playwright/app-startup-experience");
const axePath = path.join(root, "apps/ak-admin/node_modules/axe-core/axe.min.js");
const image = await readFile(path.join(root, "output/playwright/admin-login.zh-CN.768.png"));
await mkdir(output, { recursive: true });

const tenantId = "523e4567-e89b-12d3-a456-426614174000";
const appId = "623e4567-e89b-12d3-a456-426614174000";
const userId = "423e4567-e89b-12d3-a456-426614174000";
const iconFileId = "a23e4567-e89b-12d3-a456-426614174001";
const permissions = ["app.application.read", "app.application.update", "app.onboarding.publish", "storage.file.read"];
const application = {
  id: appId, tenant_id: tenantId, appid: "__UNI__APPKERNIA", appid_pending: false, app_type: "uni_app_x",
  code: "default-app", name: "AppKernia", description: "跨平台应用开发基座", introduction: "Build once and operate safely.", remark: "Startup E2E fixture",
  status: "active", default_locale: "zh-CN", registration_enabled: true, registration_verification_mode: "email_otp",
  creator_user_id: userId, owner_type: "tenant", owner_id: tenantId, icon_file_id: iconFileId,
  managers: [userId], members: [], screenshots: [], channels: [], store_listings: [], is_default: true, lock_version: 3,
  created_at: "2026-08-05T02:00:00Z", updated_at: "2026-08-25T02:00:00Z",
  startup: {
    translations: {
      "zh-CN": { display_name: "AppKernia", subtitle: "安全一致的跨端应用基座" },
      "en-US": { display_name: "AppKernia", subtitle: "A consistent and secure cross-platform foundation" },
    },
    onboarding_enabled: true,
    draft_slides: [
      { id: "b23e4567-e89b-12d3-a456-426614174001", position: 0, assets: {
        "zh-CN": { file_id: "a23e4567-e89b-12d3-a456-426614174002", accessibility_label: "第一张中文启动介绍" },
        "en-US": { file_id: "a23e4567-e89b-12d3-a456-426614174003", accessibility_label: "First English onboarding image" },
      } },
      { id: "b23e4567-e89b-12d3-a456-426614174002", position: 1, assets: {
        "zh-CN": { file_id: "a23e4567-e89b-12d3-a456-426614174004", accessibility_label: "第二张中文启动介绍" },
        "en-US": { file_id: "a23e4567-e89b-12d3-a456-426614174005", accessibility_label: "Second English onboarding image" },
      } },
    ],
    published_version: 3, published_at: "2026-08-24T02:00:00Z", draft_changed: true,
  },
};

function envelope(data) {
  return JSON.stringify({ code: "OK", message: "OK", data, request_id: "startup-e2e" });
}

async function routeHandler(route) {
  const request = route.request();
  const url = new URL(request.url());
  const pathname = url.pathname;
  const method = request.method();
  if (pathname.endsWith("/auth/public-config")) return route.fulfill({ status: 200, contentType: "application/json", body: envelope({ locale: "zh-CN", default_locale: "zh-CN", supported_locales: ["zh-CN", "en-US"], feature_flags: {}, settings: {} }) });
  if (pathname.endsWith("/auth/login")) return route.fulfill({ status: 200, contentType: "application/json", body: envelope({ access_token: "e2e", token_type: "Bearer", expires_in: 900, csrf_token: "csrf-e2e-value-long-enough" }) });
  if (pathname.endsWith("/auth/context")) return route.fulfill({ status: 200, contentType: "application/json", body: envelope({ user: { id: userId, email: "e2e@app.test", display_name: "E2E", locale: "zh-CN", time_zone: "UTC", avatar_url: null }, active_tenant: { id: tenantId, name: "E2E", code: "e2e" }, available_tenants: [], roles: [], permissions, menus: [], feature_flags: {}, menu_revision: 1, permission_revision: 1, server_time: "2026-08-25T02:00:00Z" }) });
  if (pathname.endsWith("/me") && method === "PATCH") return route.fulfill({ status: 200, contentType: "application/json", body: envelope({ id: userId, email: "e2e@app.test", display_name: "E2E", locale: request.postDataJSON().locale, time_zone: "UTC", avatar_url: null }) });
  if (pathname.endsWith("/dictionaries/system.language")) {
    const english = (request.headers()["accept-language"] ?? "").startsWith("en-US");
    return route.fulfill({ status: 200, contentType: "application/json", body: envelope({ code: "system.language", locale: english ? "en-US" : "zh-CN", extension_policy: "fixed", items: [
      { value: "zh-CN", label: english ? "Simplified Chinese" : "简体中文", is_default: true, extra: {} },
      { value: "en-US", label: "English", is_default: false, extra: {} },
    ] }) });
  }
  if (pathname.endsWith("/apps") && method === "GET") return route.fulfill({ status: 200, contentType: "application/json", body: envelope({ items: [application], total: 1 }) });
  if (pathname.endsWith(`/apps/${appId}`) && method === "GET") return route.fulfill({ status: 200, contentType: "application/json", body: envelope(application) });
  if (pathname.includes("/files/") && pathname.endsWith("/content")) return route.fulfill({ status: 200, contentType: "image/png", body: image });
  return route.fulfill({ status: 200, contentType: "application/json", body: envelope({}) });
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function assertNoOverflow(page) {
  const size = await page.evaluate(() => ({ scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth }));
  assert(size.scrollWidth <= size.clientWidth, `page overflow: ${JSON.stringify(size)}`);
}

async function runAxe(page) {
  const result = await page.evaluate(async () => await globalThis.axe.run({ exclude: [[".ant-table-measure-row"]] }, { resultTypes: ["violations"] }));
  const severe = result.violations.filter((item) => item.impact === "critical" || item.impact === "serious");
  assert(severe.length === 0, `axe serious/critical: ${JSON.stringify(severe)}`);
  return result.violations.map((item) => ({ id: item.id, impact: item.impact, nodes: item.nodes.length }));
}

async function navigate(page, pathname) {
  await page.evaluate((value) => { window.history.pushState({}, "", value); window.dispatchEvent(new PopStateEvent("popstate")); }, pathname);
}

const evidence = {};
const consoleErrors = [];
const browser = await chromium.launch();
const context = await browser.newContext({ locale: "zh-CN", viewport: { width: 1440, height: 900 }, colorScheme: "light" });
const page = await context.newPage();
page.setDefaultTimeout(15000);
await page.route("**/admin-api/**", routeHandler);
page.on("console", (message) => { if (message.type() === "error") consoleErrors.push(message.text()); });

await page.goto(`${base}/login`, { waitUntil: "domcontentloaded" });
await page.locator("#login-email").fill("e2e@app.test");
await page.locator("#login-password").fill("password");
await page.locator('button[type="submit"]').click();
await page.waitForURL("**/dashboard**");
await navigate(page, "/app/applications");
await page.getByRole("heading", { name: "应用管理" }).waitFor();
await page.locator("button:has-text('编 辑')").first().click();
let drawer = page.getByRole("dialog");
await drawer.waitFor();
await drawer.getByText("启动体验", { exact: true }).scrollIntoViewIfNeeded();
assert(await drawer.getByText("已发布版本 v3", { exact: true }).isVisible(), "published version missing");
assert(await drawer.getByRole("button", { name: "发布新版本" }).isEnabled(), "publish action disabled");
const zhDescriptions = drawer.getByLabel("简体中文 无障碍说明");
assert(await zhDescriptions.count() === 2, "bilingual slide inputs missing");
assert(await zhDescriptions.nth(0).inputValue() === "第一张中文启动介绍", "initial order is wrong");
await drawer.getByRole("button").filter({ hasText: /下\s*移/ }).first().focus();
await page.keyboard.press("Enter");
assert(await zhDescriptions.nth(0).inputValue() === "第二张中文启动介绍", "keyboard move down failed");
await drawer.getByRole("button").filter({ hasText: /上\s*移/ }).nth(1).focus();
await page.keyboard.press("Enter");
assert(await zhDescriptions.nth(0).inputValue() === "第一张中文启动介绍", "keyboard move up failed");
await page.addScriptTag({ path: axePath });

for (const width of [375, 768, 1440]) {
  await page.setViewportSize({ width, height: width === 375 ? 812 : 900 });
  await drawer.getByText("启动体验", { exact: true }).scrollIntoViewIfNeeded();
  await assertNoOverflow(page);
  evidence[`startup.drawer.zh-CN.light.${width}`] = { violations: await runAxe(page) };
  await page.screenshot({ path: path.join(output, `startup.drawer.zh-CN.light.${width}.png`), fullPage: true });
}
await page.keyboard.press("Escape");
await drawer.waitFor({ state: "hidden" });

await page.getByLabel("显示语言").click();
await page.getByRole("menuitem", { name: "English" }).click();
await page.getByRole("heading", { name: "Application management" }).waitFor();
await page.getByRole("button", { name: "Edit" }).first().click();
drawer = page.getByRole("dialog");
await drawer.waitFor();
for (const width of [375, 768, 1440]) {
  await page.setViewportSize({ width, height: width === 375 ? 812 : 900 });
  await drawer.getByText("Startup experience", { exact: true }).scrollIntoViewIfNeeded();
  assert(await drawer.getByLabel("English Accessibility description").count() === 2, "English asset descriptions missing");
  await assertNoOverflow(page);
  evidence[`startup.drawer.en-US.light.${width}`] = { violations: await runAxe(page) };
  await page.screenshot({ path: path.join(output, `startup.drawer.en-US.light.${width}.png`), fullPage: true });
}

evidence.console = { unexpected_errors: consoleErrors };
evidence.coverage = { locales: ["zh-CN", "en-US"], viewports: [375, 768, 1440], keyboard_ordering: true };
assert(consoleErrors.length === 0, `unexpected console errors: ${JSON.stringify(consoleErrors)}`);
await writeFile(path.join(output, "e2e-results.json"), `${JSON.stringify(evidence, null, 2)}\n`, "utf8");
await context.close();
await browser.close();
console.log("app-startup-e2e:passed");
