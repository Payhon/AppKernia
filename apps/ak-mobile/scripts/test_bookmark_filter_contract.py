#!/usr/bin/env python3
"""Cross-layer contract tests for the mobile bookmark type filters."""

from __future__ import annotations

import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]


class BookmarkFilterContractTests(unittest.TestCase):
    def read(self, path: Path) -> str:
        return path.read_text(encoding="utf-8")

    def test_page_sends_selected_type_and_guards_stale_responses(self) -> None:
        page = self.read(ROOT / "pages/bookmarks/index.uvue")
        for expected in (
            "types:['','article','gallery','video']",
            "contentType:this.selectedType.length>0?this.selectedType:null",
            "if(this.selectedType===x)return",
            "if(version!==requestVersion)return",
            "active.cancel()",
            "this.state='loading'",
        ):
            self.assertIn(expected, page)

    def test_tabs_expose_selected_semantics(self) -> None:
        page = self.read(ROOT / "pages/bookmarks/index.uvue")
        profile = self.read(ROOT / "pages/profile/index.uvue")
        section_list = self.read(ROOT / "components/section-list/section-list.uvue")
        self.assertIn('role="tablist"', page)
        self.assertIn('role="tab"', page)
        self.assertIn(':aria-selected="selectedType === type"', page)
        self.assertIn(':aria-label="typeName(type)"', page)
        self.assertIn(
            'role="button" :aria-label="t(\'profile.bookmarks\', null)" @click="protectedGo(\'/pages/bookmarks/index\')"',
            profile,
        )
        self.assertEqual(section_list.count('role="button" :aria-label="item.title"'), 2)

    def test_repository_and_openapi_forward_type(self) -> None:
        repository = self.read(ROOT / "src/features/articles/infrastructure/http-article-repository.uts")
        openapi = self.read(REPO / "server/openapi/openapi.yaml")
        endpoint = openapi.index("/api/v1/me/content-bookmarks:")
        endpoint_end = openapi.index("/api/v1/me/content-bookmarks/{article_id}:", endpoint)
        self.assertIn("p.set('type', query.contentType)", repository)
        self.assertIn("{name: type, in: query", openapi[endpoint:endpoint_end])

    def test_server_bookmark_query_applies_content_type(self) -> None:
        source = self.read(REPO / "server/internal/modules/content/repository/information.go")
        start = source.index("func bookmarkListQuery(")
        query = source[start:]
        for expected in (
            'if f.ContentType != ""',
            'add("a.content_type=$%d", f.ContentType)',
            'if f.Query != ""',
            'if f.Cursor != ""',
            "f.Limit+1",
        ):
            self.assertIn(expected, query)


if __name__ == "__main__":
    unittest.main()
