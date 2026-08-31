#!/usr/bin/env python3
"""Source-level scanner contract tests for uni-app x builds.

HBuilderX is the executable UTS compiler gate. These checks protect the
cross-platform semantics that are otherwise easy to regress mechanically.
"""

from pathlib import Path
import re
import unittest


ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class ScannerContractTests(unittest.TestCase):
    def test_native_adapters_are_camera_only_qr_and_barcode(self) -> None:
        for platform in ("app-android", "app-ios", "app-harmony"):
            source = read(f"uni_modules/ak-scanner/utssdk/{platform}/index.uts")
            self.assertIn("onlyFromCamera:true", source)
            self.assertIn("scanType:['qrCode','barCode']", source)
            self.assertIn("cancelled()", source)
            self.assertNotIn("chooseImage", source)

    def test_ios_simulator_fails_before_calling_native_scanner(self) -> None:
        source = read("uni_modules/ak-scanner/utssdk/app-ios/index.uts")
        bridge = read("uni_modules/ak-scanner/utssdk/app-ios/hybrid.swift")
        guard = source.index("AkScannerIOSBridge.isSimulator()")
        native_call = source.index("uni.scanCode")
        self.assertLess(guard, native_call)
        self.assertIn("failed('scanner_unavailable')", source[guard:native_call])
        self.assertIn("#if targetEnvironment(simulator)", bridge)

    def test_stable_result_event_handler_and_resolution_types_exist(self) -> None:
        interface = read("uni_modules/ak-scanner/utssdk/interface.uts")
        for token in (
            "scanId",
            "rawValue",
            "format",
            "source",
            "scannedAt",
            "'captured' | 'parsed' | 'resolved' | 'cancelled' | 'failed'",
            "'consumed' | 'open_webview' | 'present_result'",
            "readonly priority: number",
            "readonly canHandle:",
            "readonly handle:",
        ):
            self.assertIn(token, interface)

    def test_coordinator_is_single_flight_ordered_and_disposable(self) -> None:
        source = read("src/features/scanner/application/scanner-coordinator.uts")
        self.assertIn("if(this.busy){", source)
        self.assertIn("this.busy=true", source)
        self.assertIn("right.priority-left.priority", source)
        self.assertIn("value.id!=handler.id", source)
        self.assertIn("value!=listener", source)
        captured = source.index("new AkScanEvent('captured'")
        parsed = source.index("new AkScanEvent('parsed'")
        resolved = source.index("new AkScanEvent('resolved'")
        handlers = source.index("for(let index=0;index<this.handlers.length")
        webview = source.index("if(this.webViewEnabled)")
        fallback = source.index("return 'present_result'")
        self.assertLess(captured, parsed)
        self.assertLess(parsed, resolved)
        self.assertLess(handlers, webview)
        self.assertLess(webview, fallback)

    def test_url_policy_guards_spoofing_boundaries(self) -> None:
        source = read("src/features/scanner/domain/url-policy.uts")
        required = (
            "startsWith('https://')",
            "authority.indexOf('@')>=0",
            "authority.substring(colon+1)!='443'",
            "host=='localhost'",
            "host!=suffix",
            "host.endsWith('.'+suffix)",
        )
        for token in required:
            self.assertIn(token, source)
        self.assertRegex(source, re.compile(r"let numeric=true.*if\(numeric\)return null", re.S))

    def test_webview_uses_single_use_token_and_rechecks_navigation(self) -> None:
        store = read("src/features/scanner/application/webview-ticket-store.uts")
        page = read("pages/scanner/webview/index.uvue")
        coordinator = read("src/features/scanner/application/scanner-coordinator.uts")
        self.assertIn("this.tickets.delete(token)", store)
        self.assertIn("now-value.createdAt>60000", store)
        self.assertIn("?token='+encodeURIComponent(token)", coordinator)
        self.assertNotIn("?url=", coordinator)
        self.assertIn("webViewTicketStore.consume(decoded)", page)
        self.assertIn('@loading="guardLoading"', page)
        self.assertIn('@load="guardLoad"', page)
        self.assertIn("allowedHttpsTarget(value,current.allowedHostPatterns)", page)
        self.assertNotIn("@message", page)
        self.assertNotIn("onMessage", page)

    def test_home_entry_is_guest_available_and_subscription_is_released(self) -> None:
        page = read("pages/home/index.uvue")
        self.assertIn('name="scan"', page)
        self.assertIn(":disabled=\"scanBusy\"", page)
        self.assertIn("scannerCoordinator.scan()", page)
        self.assertIn("subscription.dispose()", page)
        scan_method = page[page.index("scan(){") : page.index("onScanEvent")]
        self.assertNotIn("getAccessToken", scan_method)


if __name__ == "__main__":
    unittest.main()
