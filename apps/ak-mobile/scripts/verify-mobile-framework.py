#!/usr/bin/env python3
"""Fast contract checks for the mobile framework that do not require a device."""

from __future__ import annotations

import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]


def fail(message: str) -> None:
    raise SystemExit(f"FAIL: {message}")


def read_json(path: Path) -> object:
    with path.open(encoding="utf-8") as file:
        return json.load(file)


def required(condition: bool, message: str) -> None:
    if not condition:
        fail(message)


def check_locales() -> None:
    zh = read_json(ROOT / "locale/zh-CN.json")
    en = read_json(ROOT / "locale/en-US.json")
    required(isinstance(zh, dict) and isinstance(en, dict), "locale catalogs must be objects")
    required(set(zh) == set(en), "zh-CN/en-US locale keys differ")
    required(all(isinstance(value, str) and value for value in zh.values()), "zh-CN contains an empty translation")
    required(all(isinstance(value, str) and value for value in en.values()), "en-US contains an empty translation")
    for key in (
        "errors.auth.unauthorized",
        "errors.auth.forbidden",
        "errors.request.cancelled",
        "errors.request.timeout",
        "errors.request.network_unavailable",
        "errors.request.conflict",
        "errors.request.validation",
        "errors.request.rate_limited",
        "errors.request.maintenance",
        "errors.request.server_error",
    ):
        required(key in zh and key in en, f"network error translation missing {key}")
    i18n_runtime = (ROOT / "src/core/i18n/ak-i18n.uts").read_text(encoding="utf-8")
    required("catalogFromEntries" in i18n_runtime, "i18n runtime does not load generated typed entries")
    required(".toMap()" not in i18n_runtime, "i18n runtime must not call UTSJSONObject methods on imported JSON")


def check_routes_and_contracts() -> None:
    pages = read_json(ROOT / "pages.json")
    registered = {item["path"] for item in pages["pages"]}
    for route in ("pages/home/index", "pages/profile/index", "pages/articles/list/index", "pages/articles/detail/index"):
        required(route in registered, f"pages.json does not register {route}")
    route_registry = read_json(REPO / "blueprint/mobile/spec/mobile-route-registry.json")
    route_ids = {item["route_key"] for item in route_registry["routes"]}
    required({"articles.list", "articles.detail"}.issubset(route_ids), "article routes are absent from mobile route registry")
    openapi = (REPO / "server/openapi/openapi.yaml").read_text(encoding="utf-8")
    for fragment in ("/api/v1/articles:", "/api/v1/articles/{slug}:", "/api/v1/me/article-bookmarks/{article_id}:", "AppArticleListResponse:"):
        required(fragment in openapi, f"OpenAPI article contract missing {fragment}")
    required("/api/v1/article-categories:" in openapi and "AppArticleCategoryListResponse:" in openapi, "OpenAPI category contract missing")


def check_safe_renderer() -> None:
    renderer = (ROOT / "src/features/articles/application/article-body-renderer.uts").read_text(encoding="utf-8")
    detail = (ROOT / "pages/articles/detail/index.uvue").read_text(encoding="utf-8")
    required("JSON.parse<Array<ArticleWireBlock>>" in renderer, "blocks are not parsed through the restricted DTO")
    required("item.type != 'heading' && item.type != 'paragraph' && item.type != 'callout'" in renderer, "block type allowlist is missing")
    required("<rich-text" not in detail and "v-html" not in detail, "detail page must not render untrusted rich content")
    required("articleBodyRenderer.render" in detail, "detail page is not using the safe body renderer")


def check_query_and_lifecycle() -> None:
    list_page = (ROOT / "pages/articles/list/index.uvue").read_text(encoding="utf-8")
    client = (ROOT / "src/core/network/ak-http-client.uts").read_text(encoding="utf-8")
    for fragment in ("onLoad() { this.reload() }", "@scrolltolower=\"loadMore\"", "cursor: cursor", "repository.setBookmark", "const listTask = pendingList", "setTimeout", "clearTimeout", "repository.listCategories", "category.slug"):
        required(fragment in list_page, f"article list lifecycle missing {fragment}")
    for fragment in ("Accept-Language", "Authorization", "timeout: this.config.requestTimeoutMs", "task.abort()", "errorFromStatus"):
        required(fragment in client, f"HTTP client missing {fragment}")
    home = (ROOT / "pages/home/index.uvue").read_text(encoding="utf-8")
    for fragment in ("onLoad() { this.reload() }", "profiles.profile", "profiles.unreadCount", "articles.list", "onUnload() { this.cancelRequests() }", "featured: true", "[['count', minutes.toString()]]"):
        required(fragment in home, f"home data lifecycle missing {fragment}")


def check_runtime_wiring() -> None:
    app = (ROOT / "App.uvue").read_text(encoding="utf-8")
    bootstrap_page = (ROOT / "pages/bootstrap/index.uvue").read_text(encoding="utf-8")
    bootstrap = (ROOT / "src/core/bootstrap/app-bootstrap.uts").read_text(encoding="utf-8")
    manifest = read_json(ROOT / "manifest.json")
    runtime = manifest.get("akRuntime") if isinstance(manifest, dict) else None
    required("initializeAppRuntime" not in app, "App must not duplicate bootstrap runtime initialization")
    required("initializeAppRuntime()" in bootstrap_page and "onLoad" in bootstrap_page, "bootstrap page does not initialize runtime")
    required("articleRuntime.configure(config, appSessionStore, auth" in bootstrap, "article runtime is not configured at bootstrap")
    required(isinstance(runtime, dict) and "apiBaseUrl" in runtime, "manifest has no approved runtime API configuration")
    api_base_url = runtime.get("apiBaseUrl")
    required(isinstance(api_base_url, str), "manifest runtime API base URL must be a string")
    required(api_base_url == "" or api_base_url.startswith("http://127.0.0.1:"), "manifest must not embed a production API base URL")
    secure_port = (ROOT / "src/core/stores/secure-session-storage-port.uts").read_text(encoding="utf-8")
    required("SecureSessionStoragePort" in secure_port and "uni.setStorage" not in secure_port, "session persistence must use a secure-storage port")
    runtime = (ROOT / "src/core/stores/session-runtime.uts").read_text(encoding="utf-8")
    test_adapter = (ROOT / "src/core/stores/in-memory-secure-session-storage.uts").read_text(encoding="utf-8")
    required("restore(" in runtime and "signOut(" in runtime and "storage.write" in runtime, "session runtime lacks restore/persist/logout lifecycle")
    required("Test-only adapter" in test_adapter and "uni.setStorage" not in test_adapter, "in-memory test adapter is missing or insecure")
    auth_repository = (ROOT / "src/features/auth/auth-repository.uts").read_text(encoding="utf-8")
    auth_service = (ROOT / "src/features/auth/mobile-auth-service.uts").read_text(encoding="utf-8")
    auth_runtime = (ROOT / "src/features/auth/auth-runtime.uts").read_text(encoding="utf-8")
    auth_context = (ROOT / "src/core/stores/auth-context-store.uts").read_text(encoding="utf-8")
    routes = (ROOT / "src/core/navigation/app-routes.uts").read_text(encoding="utf-8")
    profile = (ROOT / "pages/profile/index.uvue").read_text(encoding="utf-8")
    notifications_preferences = (ROOT / "src/features/preferences/infrastructure/http-preferences-repository.uts").read_text(encoding="utf-8")
    about = (ROOT / "pages/about/index.uvue").read_text(encoding="utf-8")
    asset_loader = (ROOT / "src/features/articles/infrastructure/article-asset-loader.uts").read_text(encoding="utf-8")
    detail = (ROOT / "pages/articles/detail/index.uvue").read_text(encoding="utf-8")
    refresh_port = (ROOT / "src/core/network/refreshing-http-port.uts").read_text(encoding="utf-8")
    refresh_coordinator = (ROOT / "src/core/network/refresh-coordinator.uts").read_text(encoding="utf-8")
    for fragment in ("/auth/login/password", "X-AK-Device-Key", "/auth/token/refresh", "/auth/logout", "/auth/context", "refresh_token"):
        required(fragment in auth_repository, f"auth repository missing {fragment}")
    required("sessions.establish" in auth_service and "refreshToken" in auth_service and "signOut(onComplete" in auth_service, "auth service does not persist rotated refresh credentials or clear logout state")
    required("authContextStore.set" in auth_runtime and "navigationContext" in auth_context and "feature_flags: UTSJSONObject" in auth_repository and "feature_flags.toMap()" in auth_repository, "runtime does not load server auth context safely")
    required("createAuthenticatedNavigationContext" not in routes and "permission: null" in routes, "routes must not manufacture self-scope permissions")
    required("authRuntime.signOut" in profile and "appSessionStore.clear" not in profile, "profile sign-out bypasses remote/local auth lifecycle")
    required("NotificationPrefs" in notifications_preferences and "locale:'zh-CN'" not in notifications_preferences, "notification endpoint must use its independent payload")
    required("// #ifdef APP-ANDROID\n" in about and "// #ifdef APP-IOS\n" in about and "/* #ifdef" not in about, "version platform must use real conditional compilation")
    for fragment in ("uni.downloadFile", "Authorization: 'Bearer ' + token", "result.statusCode == 401 && !replayed", "task.abort()", "article-assets"):
        required(fragment in asset_loader, f"authenticated article asset loader missing {fragment}")
    required("const currentTask = coverTask" in detail and "loader.load(path" in detail, "detail page does not cancel authenticated asset downloads")
    required("replayed || !this.canReplay" in refresh_port and "request.retryOnUnauthorized && request.method == 'GET'" in refresh_port and "DELETE" not in refresh_port.split("private canReplay", 1)[1], "401 replay policy must be explicitly opted-in GET-only")
    required("if (this.refreshing) return" in refresh_coordinator and "sessions.signOut" in refresh_coordinator, "refresh single-flight/session clear is absent")


def check_shared_controls_and_profile_menu() -> None:
    field = (ROOT / "components/ak-ui/ak-text-field/ak-text-field.uvue").read_text(encoding="utf-8")
    button = (ROOT / "components/ak-ui/ak-button/ak-button.uvue").read_text(encoding="utf-8")
    profile = (ROOT / "pages/profile/index.uvue").read_text(encoding="utf-8")
    notifications = (ROOT / "pages/settings/notifications/index.uvue").read_text(encoding="utf-8")
    for fragment in ("disabled: { type: Boolean", ":disabled=\"disabled\"", "ak-field__input--disabled"):
        required(fragment in field, f"text field disabled contract missing {fragment}")
    for fragment in ("variantClass", "ak-button-primary", "ak-button-secondary", "ak-button-danger", "disabled || loading"):
        required(fragment in button, f"button variant/disabled contract missing {fragment}")
    for route in ("settings.notifications", "settings.appearance", "help", "about"):
        required(f"go('{route}')" in profile, f"profile menu does not expose {route}")
    required("settings.notificationInApp" in notifications and "inApp:this.inApp" in notifications, "notification settings omit in-app preference")


def main() -> None:
    check_locales()
    check_routes_and_contracts()
    check_safe_renderer()
    check_query_and_lifecycle()
    check_runtime_wiring()
    check_shared_controls_and_profile_menu()
    print("mobile framework contract checks passed")


if __name__ == "__main__":
    main()
