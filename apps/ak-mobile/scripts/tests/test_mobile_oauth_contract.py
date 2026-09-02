import hashlib
import json
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def source(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class MobileOAuthContractTests(unittest.TestCase):
    def test_runtime_routes_do_not_gate_callback_or_connections_on_oauth(self) -> None:
        routes = source("src/core/navigation/app-routes.uts")
        self.assertIn("{ key: 'auth.oauth-callback', path: 'pages/auth/oauth-callback/index', access: 'public', permission: null, featureFlag: null }", routes)
        self.assertIn("{ key: 'profile.connections', path: 'pages/profile/connections/index', access: 'authenticated', permission: null, featureFlag: null }", routes)
        machine = json.loads(source("../../blueprint/mobile/spec/mobile-route-registry.json"))
        by_key = {item["route_key"]: item for item in machine["routes"]}
        self.assertEqual(by_key["auth.oauth-callback"]["access"], "public")
        self.assertIsNone(by_key["auth.oauth-callback"]["feature_flag"])
        self.assertEqual(by_key["profile.connections"]["required_permissions"], [])
        self.assertIsNone(by_key["profile.connections"]["feature_flag"])

    def test_provider_registry_intersects_compiled_server_and_bundled_capabilities(self) -> None:
        registry = source("src/features/auth/oauth-provider-registry.uts")
        for required in (
            "this.compiledProviders.indexOf(item.providerCode)",
            "this.snapshot.buildVariants.indexOf(this.target.buildVariant)",
            "bundled.platforms.indexOf(this.target.platform)",
            "bundled.buildVariants.indexOf(this.target.buildVariant)",
            "bundled.buildConfigHash != item.buildConfigHash",
            "bundled.configSchemaVersion != item.configSchemaVersion",
        ):
            self.assertIn(required, registry)
        self.assertNotIn("!bundled.enabled", registry)
        self.assertIn("oauthBundledSnapshotJSON", registry)
        self.assertNotIn("import bundledProviderSnapshot", registry)

    def test_consent_is_a_coordinator_boundary_not_only_a_page_condition(self) -> None:
        coordinator = source("src/features/auth/oauth-coordinator.uts")
        self.assertGreaterEqual(coordinator.count("legalConsentStore.acceptedAll()"), 3)
        self.assertIn("if (!legalConsentStore.acceptedAll()) { onSuccess([]); return }", coordinator)

    def test_login_identifier_and_step_up_requests_use_frozen_paths_and_identifier_id(self) -> None:
        repository = source("src/features/auth/login-methods-repository.uts")
        generated = source("src/generated/api/mobile-identity.uts")
        self.assertIn("'/me/login-identifiers/' + identifierType + '/challenge'", repository)
        self.assertIn("method: 'PUT', path: '/me/login-identifiers/' + identifierType", repository)
        self.assertIn("method: 'DELETE', path: '/me/login-identifiers/' + identifierType", repository)
        self.assertIn("identifier_id: identifierId", repository)
        self.assertIn("challenge_id: challengeId", repository)
        self.assertNotIn("/me/identifiers/", repository)
        self.assertNotIn("display_hint: identifier", repository)
        self.assertIn("from '../../generated/api/mobile-identity.uts'", repository)
        self.assertIn("readonly identifier_id: string", generated)
        self.assertIn("readonly challenge_id: string", generated)
        login_form = source("components/ak-ui/ak-login-method-form/ak-login-method-form.uvue")
        connections = source("pages/profile/connections/index.uvue")
        self.assertIn("/^[0-9]{6}$/", login_form)
        self.assertNotIn("{4,8}", login_form)
        self.assertGreaterEqual(connections.count(':max-length="6"'), 2)
        self.assertNotIn("{4,8}", connections)

    def test_account_deletion_is_server_driven_and_apple_reauths_before_confirm(self) -> None:
        page = source("pages/profile/account-deletion/index.uvue")
        repository = source("src/features/account-deletion/infrastructure/http-account-deletion-repository.uts")
        self.assertIn("verificationReady = !this.verificationRequired", page)
        self.assertIn("/^[0-9]{6}$/.test(this.verificationCode)", page)
        self.assertIn("oauthCoordinator.reauthorize('apple', this.reauthAccountId, 'account_delete'", page)
        self.assertIn("stepUpToken: this.reauthRequired ? this.stepUpToken : ''", page)
        self.assertIn("verification_required", repository)
        self.assertIn("reauth_account_id", repository)
        self.assertIn("step_up_token: input.stepUpToken", repository)
        generated = source("src/generated/api/mobile-account-deletion.uts")
        self.assertIn("readonly reauth_required: boolean", generated)
        self.assertIn("readonly reauth_account_id: string | null", generated)
        self.assertIn("readonly step_up_token: string", generated)

    def test_apple_first_authorization_name_is_bounded_and_non_authoritative(self) -> None:
        bridge = source("uni_modules/ak-oauth/utssdk/app-ios/hybrid.swift")
        repository = source("src/features/auth/oauth-repository.uts")
        coordinator = source("src/features/auth/oauth-coordinator.uts")
        self.assertIn("credential.fullName", bridge)
        self.assertIn("controlCharacters", bridge)
        self.assertIn("collapsed.prefix(120)", bridge)
        self.assertIn("display_name: displayName", repository)
        self.assertIn("if (flow.providerCode == 'apple') repository.callbackApple", coordinator)

    def test_unknown_historical_provider_is_not_filtered_from_account_management(self) -> None:
        methods = source("src/features/auth/login-methods-repository.uts")
        oauth = source("src/features/auth/oauth-repository.uts")
        connections = source("pages/profile/connections/index.uvue")
        self.assertIn("if (item.provider_code.length == 0) return", methods)
        self.assertIn("if (wire.provider_code.length == 0) return null", oauth)
        self.assertIn("connections.providerUnknown", connections)
        self.assertIn("oauthCoordinator.unbind(this.pendingAccountId", connections)

    def test_oauth_only_binding_uses_server_capability_and_block_reason(self) -> None:
        connections = source("pages/profile/connections/index.uvue")
        self.assertIn("item.status != 'unbound' || !item.canBind", connections)
        self.assertIn("item.blockReason == 'step_up_method_required'", connections)
        self.assertIn("connections.stepUp.bindMethodRequired", connections)

    def test_github_return_is_bundled_allowlisted_and_malformed_encoding_fails_closed(self) -> None:
        coordinator = source("src/features/auth/oauth-coordinator.uts")
        self.assertIn("registry.githubReturnURIs().indexOf(base)", coordinator)
        self.assertIn("try { decodedKey = decodeURIComponent", coordinator)
        self.assertIn("catch (_error) { return null }", coordinator)
        self.assertIn("parts.length != 3", coordinator)
        self.assertIn("oauth.browser_cancelled", coordinator)

    def test_installation_key_is_ready_and_stable_before_cold_callback_dispatch(self) -> None:
        runtime = source("src/features/auth/auth-runtime.uts")
        bootstrap = source("src/core/bootstrap/app-bootstrap.uts")
        session_storage = source("src/core/stores/ak-secure-session-storage.uts")
        installation_storage = source("src/core/stores/ak-secure-installation-storage.uts")
        self.assertIn("installationStorage.readDeviceKey", runtime)
        self.assertIn("this.finishConfigure(value!, onReady)", runtime)
        self.assertIn("this.installationStorage.writeDeviceKey(value", runtime)
        self.assertIn("migrateSessionDeviceKeyOrCreate", runtime)
        self.assertIn("activeSessions.readCredential", runtime)
        self.assertIn("writeDeviceKey(credential.deviceKey", runtime)
        self.assertLess(runtime.index("this.applyDeviceKey(value)"), runtime.index("oauthCoordinator.configure"))
        self.assertIn("authRuntime.configure(client, sessionRuntime, () => continueAfterDeviceKey", bootstrap)
        self.assertIn("installation.device-key", installation_storage)
        self.assertNotIn("clearNamespace", session_storage)
        self.assertIn("SESSION_KEYS", session_storage)
        self.assertIn("installation.device-key", installation_storage)
        self.assertNotIn("installation.device-key", session_storage)
        self.assertNotIn("uni.setStorage", runtime + installation_storage)

    def test_oauth_repositories_consume_openapi_generated_wire_types(self) -> None:
        oauth = source("src/features/auth/oauth-repository.uts")
        methods = source("src/features/auth/login-methods-repository.uts")
        auth = source("src/features/auth/auth-repository.uts")
        for repository in (oauth, methods, auth):
            self.assertIn("generated/api/mobile-identity.uts", repository)
        self.assertNotIn("type OAuthProviderWire =", oauth)
        self.assertNotIn("type LoginMethodsWire =", methods)

    def test_provider_brand_assets_are_local_sourced_and_have_reviewed_hashes(self) -> None:
        expected = {
            "apple-black.png": "a444cd18e9af6429ea2a8a5e6f59febc7e6286132a2b760e0a6de56d812ad3b6",
            "apple-white.png": "80b313b9d4bb2963bf9e0dddbf66226c925562c82e5560101aa83172b300be33",
            "github-black.png": "2a2f5cbcc74c7fa83c40127dd8b0e42c23f1157131aebe28492aa1ac27bbdc6d",
            "github-white.png": "0d4c235fef9efec54174a7c005fc0fe0ce2d63d35c21898ab5148587111397a9",
            "google-light.png": "707f430da8d0de13091baa30729a49475cd84af6b1d849b2a76aeabe0c093397",
            "google-dark.png": "2589cb7bab12af32a9bd8ff01abfa4a71789f299d7f8dd52855b08f834160afb",
            "wechat.svg": "754d7e5ee7278786973912906a1412d5b4cf61795f75f75459bdaa353e286ed7",
        }
        for filename, digest in expected.items():
            self.assertEqual(hashlib.sha256((ROOT / "static/login-providers" / filename).read_bytes()).hexdigest(), digest)
        assets = source("src/features/auth/oauth-provider-assets.uts")
        connections = source("pages/profile/connections/index.uvue")
        self.assertIn("/static/login-providers/apple-black.png", assets)
        self.assertIn("/static/ak-icons/profile.svg", assets)
        self.assertNotIn("providerMark", connections)
        provider_list = source("components/ak-ui/ak-login-provider-list/ak-login-provider-list.uvue")
        self.assertIn(".provider-action{width:48px;height:48px;min-height:48px", provider_list)
        self.assertNotIn("provider-label", provider_list)
        self.assertNotIn("provider-state", provider_list)


if __name__ == "__main__":
    unittest.main()
