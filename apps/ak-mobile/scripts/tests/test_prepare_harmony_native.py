import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "prepare-harmony-native.py"
SPEC = importlib.util.spec_from_file_location("prepare_harmony_native", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class HarmonyOAuthOverlayTests(unittest.TestCase):
    def test_unrelated_view_data_skill_is_preserved(self) -> None:
        unrelated = {
            "entities": ["entity.system.browsable"],
            "actions": ["ohos.want.action.viewData"],
            "uris": [{"scheme": "https", "host": "content.example.com", "path": "/articles"}],
        }
        previous = [{"actions": ["wxentity.action.open"], "uris": [{"scheme": "weixin"}]}]
        current = [{"actions": ["ohos.want.action.viewData"], "uris": [{"scheme": "https", "host": "login.example.com", "path": "/oauth/return"}]}]
        merged = MODULE.merge_oauth_skills([unrelated] + previous, previous, current)
        self.assertEqual(merged, [unrelated] + current)

    def test_repeated_merge_is_idempotent(self) -> None:
        home = {"actions": ["action.system.home"]}
        current = [{"actions": ["wxentity.action.open"], "uris": [{"scheme": "weixin"}]}]
        once = MODULE.merge_oauth_skills([home], [], current)
        twice = MODULE.merge_oauth_skills(once, current, current)
        self.assertEqual(twice, once)

    def test_harmony_wechat_uses_documented_dcloud_open_sdk_boundary(self) -> None:
        root = SCRIPT.parents[1]
        source = (root / "uni_modules/ak-oauth/utssdk/app-harmony/index.uts").read_text(encoding="utf-8")
        readme = (root / "uni_modules/ak-oauth/README.md").read_text(encoding="utf-8")
        self.assertIn("provider: 'weixin'", source)
        self.assertIn("onlyAuthorize: true", source)
        self.assertIn("@tencent/wechat_open_sdk", readme)

    def test_partial_module_manifest_is_not_used_as_hbuilder_overlay(self) -> None:
        root = SCRIPT.parents[1]
        self.assertFalse((root / "harmony-configs/entry/src/main/module.json5").exists())
        prepare = SCRIPT.read_text(encoding="utf-8")
        self.assertIn('item.get("name") == "EntryAbility"', prepare)
        self.assertNotIn("ENTRY_SOURCE", prepare)


if __name__ == "__main__":
    unittest.main()
