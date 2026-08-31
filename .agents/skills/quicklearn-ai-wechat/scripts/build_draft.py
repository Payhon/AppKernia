#!/usr/bin/env python3
"""Build a deterministic, WeChat-safe draft bundle from Markdown."""

from __future__ import annotations

import argparse
import hashlib
import html
import json
import re
import shutil
import sys
from pathlib import Path
from typing import Any


CONTAINER_STYLE = (
    "max-width:100%;box-sizing:border-box;padding:0 8px;"
    "font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC',sans-serif;"
    "font-size:16px;line-height:1.85;color:#303133;word-break:break-word;"
)
PARAGRAPH_STYLE = "margin:0 0 20px 0;font-size:16px;line-height:1.85;color:#303133;"
IMAGE_TOKEN = re.compile(
    r"^\{\{IMAGE:([a-z0-9][a-z0-9-]{2,63})\}\}$",
    re.MULTILINE,
)
BODY_IMAGE_SPEC = re.compile(r"^([a-z0-9][a-z0-9-]{2,63})=(.+)$")


class BuildError(RuntimeError):
    pass


def inline_markdown(text: str) -> str:
    escaped = html.escape(text, quote=True)
    escaped = re.sub(
        r"\[([^\]]+)\]\((https?://[^\s)]+)\)",
        r'<span style="color:#576b95;">\1（\2）</span>',
        escaped,
    )
    escaped = re.sub(
        r"`([^`]+)`",
        r'<code style="padding:2px 5px;border-radius:4px;background:#f3f5f7;'
        r'color:#c7254e;font-family:SFMono-Regular,Consolas,monospace;font-size:14px;">\1</code>',
        escaped,
    )
    escaped = re.sub(
        r"\*\*(.+?)\*\*",
        r'<strong style="font-weight:650;color:#111827;">\1</strong>',
        escaped,
    )
    return escaped


def flush_paragraph(buffer: list[str], output: list[str]) -> None:
    if not buffer:
        return
    text = " ".join(part.strip() for part in buffer).strip()
    if text:
        output.append(f'<p style="{PARAGRAPH_STYLE}">{inline_markdown(text)}</p>')
    buffer.clear()


def make_digest(parts: list[str]) -> str:
    text = re.sub(r"[*_`#>\[\]()]", "", " ".join(parts))
    text = re.sub(r"\s+", " ", text).strip()
    first_sentence = re.match(r"^(.+?[。！？!?])(?:\s|$)", text)
    candidate = first_sentence.group(1) if first_sentence else text
    clipped = candidate[:54].rstrip()
    if len(candidate) > 54:
        clipped = re.sub(r"[A-Za-z0-9_-]+$", "", clipped).rstrip()
    return clipped


def render_markdown(markdown: str) -> tuple[str, str, str]:
    lines = markdown.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    title = ""
    for line in lines:
        if line.startswith("# "):
            title = line[2:].strip()
            break
        if line.strip():
            break
    if not title:
        raise BuildError("article must start with a level-one Markdown title")
    if len(title) > 32:
        raise BuildError("article title exceeds 32 characters")

    output = [f'<section style="{CONTAINER_STYLE}">']
    paragraph: list[str] = []
    list_type: str | None = None
    in_code = False
    code_lines: list[str] = []
    code_language = ""
    title_consumed = False
    digest_parts: list[str] = []

    def close_list() -> None:
        nonlocal list_type
        if list_type:
            output.append(f"</{list_type}>")
            list_type = None

    for raw in lines:
        stripped = raw.strip()
        if stripped.startswith("```"):
            flush_paragraph(paragraph, output)
            close_list()
            if not in_code:
                in_code = True
                code_language = stripped[3:].strip()
                code_lines = []
            else:
                language_label = html.escape(code_language, quote=True)
                code = html.escape("\n".join(code_lines), quote=False)
                label = (
                    f'<p style="margin:0 0 8px;color:#9ca3af;font-size:12px;">{language_label}</p>'
                    if language_label
                    else ""
                )
                output.append(
                    '<section style="margin:20px 0;padding:14px 16px;border-radius:8px;'
                    'background:#111827;overflow-wrap:anywhere;">'
                    f"{label}<pre style=\"margin:0;white-space:pre-wrap;word-break:break-word;\">"
                    f'<code style="color:#e5e7eb;font-family:SFMono-Regular,Consolas,monospace;'
                    f'font-size:13px;line-height:1.65;">{code}</code></pre></section>'
                )
                in_code = False
            continue
        if in_code:
            code_lines.append(raw)
            continue
        if not stripped:
            flush_paragraph(paragraph, output)
            close_list()
            continue
        if stripped.startswith("# "):
            if not title_consumed and stripped[2:].strip() == title:
                title_consumed = True
                continue
            flush_paragraph(paragraph, output)
            close_list()
            output.append(
                '<h2 style="margin:32px 0 16px;padding-left:12px;border-left:4px solid #2563eb;'
                f'font-size:21px;line-height:1.45;color:#111827;">{inline_markdown(stripped[2:])}</h2>'
            )
            continue
        if stripped.startswith("## "):
            flush_paragraph(paragraph, output)
            close_list()
            output.append(
                '<h2 style="margin:32px 0 16px;padding-left:12px;border-left:4px solid #2563eb;'
                f'font-size:21px;line-height:1.45;color:#111827;">{inline_markdown(stripped[3:])}</h2>'
            )
            continue
        if stripped.startswith("### "):
            flush_paragraph(paragraph, output)
            close_list()
            output.append(
                f'<h3 style="margin:24px 0 12px;font-size:18px;line-height:1.5;color:#1f2937;">'
                f"{inline_markdown(stripped[4:])}</h3>"
            )
            continue
        image_match = re.fullmatch(r"\{\{IMAGE:([a-z0-9][a-z0-9-]{2,63})\}\}", stripped)
        if image_match:
            flush_paragraph(paragraph, output)
            close_list()
            output.append(stripped)
            continue
        if stripped == "---":
            flush_paragraph(paragraph, output)
            close_list()
            output.append('<hr style="margin:32px 0;border:0;border-top:1px solid #e5e7eb;"/>')
            continue
        if stripped.startswith("> "):
            flush_paragraph(paragraph, output)
            close_list()
            text = stripped[2:].strip()
            digest_parts.append(text)
            output.append(
                '<blockquote style="margin:20px 0;padding:14px 18px;border-left:4px solid #60a5fa;'
                'background:#eff6ff;color:#374151;">'
                f'<p style="margin:0;font-size:15px;line-height:1.75;">{inline_markdown(text)}</p>'
                "</blockquote>"
            )
            continue
        unordered = re.match(r"^-\s+(.+)$", stripped)
        ordered = re.match(r"^\d+[.)]\s+(.+)$", stripped)
        if unordered or ordered:
            flush_paragraph(paragraph, output)
            wanted = "ul" if unordered else "ol"
            if list_type != wanted:
                close_list()
                output.append(
                    f'<{wanted} style="margin:16px 0 20px;padding-left:24px;color:#303133;">'
                )
                list_type = wanted
            item_text = (unordered or ordered).group(1)
            digest_parts.append(item_text)
            output.append(
                f'<li style="margin:8px 0;font-size:16px;line-height:1.75;">'
                f"{inline_markdown(item_text)}</li>"
            )
            continue
        close_list()
        paragraph.append(stripped)
        digest_parts.append(stripped)

    if in_code:
        raise BuildError("article contains an unclosed fenced code block")
    flush_paragraph(paragraph, output)
    close_list()
    output.append("</section>")
    rendered = "\n".join(output) + "\n"
    if len(rendered) >= 20000:
        raise BuildError("rendered HTML exceeds the WeChat 20,000-character limit")
    if len(rendered.encode("utf-8")) >= 1024 * 1024:
        raise BuildError("rendered HTML exceeds the WeChat 1 MiB limit")
    return title, make_digest(digest_parts), rendered


def parse_body_images(specs: list[str], article: str, output_dir: Path) -> list[dict[str, str]]:
    result: list[dict[str, str]] = []
    seen: set[str] = set()
    images_dir = output_dir / "images"
    for spec in specs:
        match = BODY_IMAGE_SPEC.fullmatch(spec)
        if not match:
            raise BuildError(f"invalid --body-image value: {spec}")
        image_id, raw_path = match.groups()
        if image_id in seen:
            raise BuildError(f"duplicate body image id: {image_id}")
        seen.add(image_id)
        placeholder = f"{{{{IMAGE:{image_id}}}}}"
        if article.count(placeholder) != 1:
            raise BuildError(f"article must contain exactly one {placeholder}")
        source = Path(raw_path).expanduser().resolve(strict=True)
        if source.suffix.lower() not in {".jpg", ".jpeg", ".png"}:
            raise BuildError(f"body image must be JPG or PNG: {source}")
        images_dir.mkdir(parents=True, exist_ok=True)
        destination = images_dir / f"{image_id}{source.suffix.lower()}"
        shutil.copy2(source, destination)
        result.append({"id": image_id, "path": str(destination.relative_to(output_dir)), "alt": image_id})

    article_ids = set(IMAGE_TOKEN.findall(article))
    if article_ids != seen:
        raise BuildError(
            f"article image placeholders {sorted(article_ids)} do not match supplied images {sorted(seen)}"
        )
    return result


def build(args: argparse.Namespace) -> dict[str, Any]:
    article_path = args.article.expanduser().resolve(strict=True)
    article = article_path.read_text(encoding="utf-8")
    title, automatic_digest, rendered = render_markdown(article)
    digest = args.digest.strip() if args.digest else automatic_digest
    if len(digest) > 120:
        raise BuildError("digest exceeds 120 characters")
    if len(args.author) > 16:
        raise BuildError("author exceeds 16 characters")

    output_dir = args.output_dir.expanduser().resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    body_images = parse_body_images(args.body_image, article, output_dir)

    cover_source = (
        args.cover.expanduser().resolve(strict=True)
        if args.cover
        else Path(__file__).resolve().parents[1] / "assets" / "default-cover.jpg"
    )
    if not cover_source.is_file() or cover_source.suffix.lower() not in {".jpg", ".jpeg", ".png"}:
        raise BuildError(f"cover must be an existing JPG or PNG file: {cover_source}")
    cover_name = f"cover{cover_source.suffix.lower()}"
    cover_destination = output_dir / cover_name
    shutil.copy2(cover_source, cover_destination)

    html_path = output_dir / "layout.html"
    html_path.write_text(rendered, encoding="utf-8")
    manifest = {
        "schema_version": "1.0",
        "title": title,
        "author": args.author,
        "digest": digest,
        "content_html": html_path.name,
        "content_source_url": args.source_url,
        "cover": cover_name,
        "body_images": body_images,
        "need_open_comment": 1,
        "only_fans_can_comment": 0,
        "article_sha256": hashlib.sha256(article.encode("utf-8")).hexdigest(),
    }
    manifest_path = output_dir / "draft-manifest.json"
    manifest_path.write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    return {
        "ok": True,
        "title": title,
        "digest_length": len(digest),
        "html_characters": len(rendered),
        "body_image_count": len(body_images),
        "manifest": str(manifest_path),
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Build a 快学AI WeChat draft bundle")
    parser.add_argument("--article", required=True, type=Path)
    parser.add_argument("--output-dir", required=True, type=Path)
    parser.add_argument("--author", default="快学AI")
    parser.add_argument("--digest", default="")
    parser.add_argument("--source-url", default="")
    parser.add_argument("--cover", type=Path)
    parser.add_argument("--body-image", action="append", default=[])
    return parser


def main() -> int:
    try:
        result = build(build_parser().parse_args())
        print(json.dumps(result, ensure_ascii=False, sort_keys=True))
        return 0
    except (BuildError, OSError, UnicodeError) as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
