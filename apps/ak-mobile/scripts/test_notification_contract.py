#!/usr/bin/env python3
"""Static cross-layer contract tests for mobile notification experiences."""

from __future__ import annotations

import json
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]


class NotificationContractTests(unittest.TestCase):
    def read(self, path: Path) -> str:
        return path.read_text(encoding="utf-8")

    def test_read_all_api_is_wired_end_to_end(self) -> None:
        openapi = self.read(REPO / "server/openapi/openapi.yaml")
        routes = self.read(REPO / "server/internal/bootstrap/api.go")
        repository = self.read(ROOT / "src/features/notifications/infrastructure/http-notifications-repository.uts")
        self.assertIn("/api/v1/me/notifications/read-all:", openapi)
        self.assertIn("operationId: markAllMobileNotificationsRead", openapi)
        self.assertIn('group.POST("/me/notifications/read-all"', routes)
        self.assertIn("path: '/me/notifications/read-all'", repository)

    def test_home_badge_uses_unread_count_and_danger_tone(self) -> None:
        page = self.read(ROOT / "pages/home/index.uvue")
        icon = self.read(ROOT / "components/ak-ui/ak-icon-button/ak-icon-button.uvue")
        for expected in (':badge="hasUnread"', 'badge-tone="danger"', "repo.unreadCount", "onShow(){this.refreshUnread();"):
            self.assertIn(expected, page)
        self.assertIn("ak-icon-button__badge--danger", icon)
        self.assertIn("var(--ak-danger)", icon)
        self.assertIn("z-index: 2", icon)

    def test_message_center_exposes_guarded_read_all_action(self) -> None:
        page = self.read(ROOT / "pages/notifications/index.uvue")
        for expected in (
            "notifications.markAllRead",
            ':disabled="!hasUnread"',
            ':loading="markAllLoading"',
            "repository.readAll",
            "notifications.markAllReadSuccess",
            "notifications.markAllReadError",
        ):
            self.assertIn(expected, page)

    def test_notification_strings_are_bilingual(self) -> None:
        zh = json.loads(self.read(ROOT / "locale/zh-CN.json"))
        en = json.loads(self.read(ROOT / "locale/en-US.json"))
        keys = {
            "notifications.openUnread",
            "notifications.markAllRead",
            "notifications.markingAllRead",
            "notifications.markAllReadSuccess",
            "notifications.markAllReadError",
        }
        self.assertTrue(keys.issubset(zh))
        self.assertTrue(keys.issubset(en))
        self.assertEqual(set(zh), set(en))

    def test_profile_keeps_inbox_and_notification_settings_destinations_distinct(self) -> None:
        page = self.read(ROOT / "pages/profile/index.uvue")
        settings_page = self.read(ROOT / "pages/settings/notifications/index.uvue")
        self.assertIn(
            ':label="t(\'routes.notifications.title\', null)"\n          @click="protectedGo(\'/pages/notifications/index\')"',
            page,
        )
        settings_row = page.index('t("profile.notifications", null)')
        settings_target = page.rfind("/pages/settings/notifications/index", 0, settings_row)
        inbox_target = page.rfind("/pages/notifications/index", 0, settings_row)
        self.assertGreater(settings_target, inbox_target)
        self.assertIn(
            'role="button" :aria-label="t(\'profile.notifications\', null)"',
            page,
        )
        self.assertIn(
            'role="heading" :aria-label="t(\'routes.settings.notifications.title\', null)"',
            settings_page,
        )


if __name__ == "__main__":
    unittest.main()
