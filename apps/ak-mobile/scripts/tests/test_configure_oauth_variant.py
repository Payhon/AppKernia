import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "configure-oauth-variant.py"
SPEC = importlib.util.spec_from_file_location("configure_oauth_variant", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class AndroidOAuthVariantTests(unittest.TestCase):
    def test_google_variant_is_the_only_variant_with_google_dependencies(self) -> None:
        google = MODULE.oauth_config(MODULE.oauth_variant("google"))
        china = MODULE.oauth_config(MODULE.oauth_variant("china"))
        disabled = MODULE.oauth_config(MODULE.oauth_variant("disabled"))
        self.assertEqual(google["dependencies"], MODULE.GOOGLE_DEPENDENCIES)
        self.assertNotIn("dependencies", china)
        self.assertNotIn("dependencies", disabled)

    def test_manifest_variant_and_bridge_templates_are_fail_closed(self) -> None:
        self.assertIn('android:value="android_google"', MODULE.oauth_manifest("android_google"))
        self.assertIn('android:value="android_china"', MODULE.oauth_manifest("android_china"))
        china = (MODULE.PLUGIN / "hybrid-china.kt.template").read_text(encoding="utf-8")
        google = (MODULE.PLUGIN / "hybrid-google.kt.template").read_text(encoding="utf-8")
        self.assertNotIn("androidx.credentials", china)
        self.assertIn('failure("native_sdk_unavailable")', china)
        self.assertIn("CredentialManager.create", google)
        self.assertIn("GetCredentialCancellationException", google)

    def test_android_build_entrypoints_invoke_variant_generator(self) -> None:
        for filename in ("build-platform.sh", "build-custom-base.sh"):
            source = (MODULE.ROOT / "scripts" / filename).read_text(encoding="utf-8")
            self.assertIn("configure-oauth-variant.py", source)


if __name__ == "__main__":
    unittest.main()
