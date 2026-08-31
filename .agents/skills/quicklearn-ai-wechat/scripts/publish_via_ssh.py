#!/usr/bin/env python3
"""Publish a local draft bundle through the SSH-only WeChat gateway."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any


DEFAULT_HOST = "codex-huaweicloud-1-95-190-254"
DEFAULT_REMOTE_BIN = "/usr/local/bin/quicklearn-wechat"
SSH_OPTIONS = ["-o", "BatchMode=yes", "-o", "ConnectTimeout=10"]
IMAGE_ID = re.compile(r"^[a-z0-9][a-z0-9-]{2,63}$")


class PublishError(RuntimeError):
    pass


def decode_json_output(data: str, context: str) -> dict[str, Any]:
    lines = [line for line in data.splitlines() if line.strip()]
    if not lines:
        raise PublishError(f"{context} returned no JSON output")
    try:
        value = json.loads(lines[-1])
    except json.JSONDecodeError as exc:
        raise PublishError(f"{context} returned invalid JSON") from exc
    if not isinstance(value, dict):
        raise PublishError(f"{context} returned a non-object JSON response")
    return value


def run_remote(
    host: str,
    remote_bin: str,
    arguments: list[str],
    *,
    input_text: str | None = None,
) -> dict[str, Any]:
    completed = subprocess.run(
        ["ssh", *SSH_OPTIONS, host, remote_bin, *arguments],
        input=input_text,
        text=True,
        capture_output=True,
        check=False,
        timeout=120,
    )
    source = completed.stdout if completed.returncode == 0 else completed.stderr
    try:
        value = decode_json_output(source, f"remote {' '.join(arguments)}")
    except PublishError:
        if completed.returncode != 0:
            raise PublishError(
                f"remote command failed with exit {completed.returncode}: {source.strip()[:500]}"
            )
        raise
    if completed.returncode != 0 or value.get("ok") is not True:
        raise PublishError(json.dumps(value, ensure_ascii=False, sort_keys=True))
    return value


def upload_remote_file(host: str, remote_bin: str, source: Path, destination: str) -> None:
    completed = subprocess.run(
        ["ssh", *SSH_OPTIONS, host, remote_bin, "stage-put", "--path", destination],
        input=source.read_bytes(),
        capture_output=True,
        check=False,
        timeout=120,
    )
    if completed.returncode != 0:
        error = completed.stderr.decode("utf-8", errors="replace")
        raise PublishError(
            f"SSH file transfer failed with exit {completed.returncode}: {error.strip()[:500]}"
        )
    value = decode_json_output(
        completed.stdout.decode("utf-8", errors="replace"),
        "remote stage-put",
    )
    if value.get("ok") is not True:
        raise PublishError(json.dumps(value, ensure_ascii=False, sort_keys=True))


def load_manifest(path: Path) -> tuple[dict[str, Any], Path]:
    resolved = path.expanduser().resolve(strict=True)
    try:
        value = json.loads(resolved.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise PublishError(f"invalid draft manifest: {exc}") from exc
    if not isinstance(value, dict):
        raise PublishError("draft manifest must be a JSON object")
    if value.get("schema_version") != "1.0":
        raise PublishError("unsupported draft manifest schema_version")
    required = ("title", "author", "content_html", "cover", "body_images")
    for key in required:
        if key not in value:
            raise PublishError(f"draft manifest is missing {key}")
    return value, resolved.parent


def resolve_bundle_file(base: Path, raw: Any, field: str) -> Path:
    if not isinstance(raw, str) or not raw:
        raise PublishError(f"{field} must be a non-empty path")
    path = (base / raw).resolve(strict=True)
    if not path.is_file():
        raise PublishError(f"{field} is not a regular file: {path}")
    return path


def validate_manifest(manifest: dict[str, Any], base: Path) -> dict[str, Any]:
    title = manifest.get("title")
    author = manifest.get("author")
    digest = manifest.get("digest", "")
    if not isinstance(title, str) or not title.strip() or len(title.strip()) > 32:
        raise PublishError("title must contain 1-32 characters")
    if not isinstance(author, str) or not author.strip() or len(author.strip()) > 16:
        raise PublishError("author must contain 1-16 characters")
    if not isinstance(digest, str) or len(digest) > 120:
        raise PublishError("digest must contain at most 120 characters")
    html_path = resolve_bundle_file(base, manifest.get("content_html"), "content_html")
    content = html_path.read_text(encoding="utf-8")
    if not content.strip() or len(content) >= 20000 or len(content.encode("utf-8")) >= 1024 * 1024:
        raise PublishError("content_html is empty or exceeds WeChat limits")
    cover_path = resolve_bundle_file(base, manifest.get("cover"), "cover")

    body_images = manifest.get("body_images")
    if not isinstance(body_images, list):
        raise PublishError("body_images must be an array")
    resolved_images: list[dict[str, Any]] = []
    seen: set[str] = set()
    for item in body_images:
        if not isinstance(item, dict):
            raise PublishError("body image entry must be an object")
        image_id = item.get("id")
        if not isinstance(image_id, str) or not IMAGE_ID.fullmatch(image_id):
            raise PublishError("body image id is invalid")
        if image_id in seen:
            raise PublishError(f"duplicate body image id: {image_id}")
        seen.add(image_id)
        image_path = resolve_bundle_file(base, item.get("path"), f"body_images[{image_id}].path")
        alt = item.get("alt", image_id)
        if not isinstance(alt, str) or len(alt) > 160:
            raise PublishError(f"body image alt text is invalid: {image_id}")
        placeholder = f"{{{{IMAGE:{image_id}}}}}"
        if content.count(placeholder) != 1:
            raise PublishError(f"content must contain exactly one {placeholder}")
        resolved_images.append({"id": image_id, "path": image_path, "alt": alt})

    return {
        "title": title.strip(),
        "author": author.strip(),
        "digest": digest,
        "content": content,
        "content_source_url": manifest.get("content_source_url", ""),
        "cover_path": cover_path,
        "body_images": resolved_images,
        "need_open_comment": bool(manifest.get("need_open_comment", 1)),
        "only_fans_can_comment": bool(manifest.get("only_fans_can_comment", 0)),
    }


def publish(args: argparse.Namespace) -> dict[str, Any]:
    manifest, base = load_manifest(args.manifest)
    bundle = validate_manifest(manifest, base)
    if args.dry_run:
        return {
            "ok": True,
            "dry_run": True,
            "title": bundle["title"],
            "html_characters": len(bundle["content"]),
            "body_image_count": len(bundle["body_images"]),
            "cover": str(bundle["cover_path"]),
        }

    staging_path: str | None = None
    try:
        staging = run_remote(args.host, args.remote_bin, ["stage-create"])
        staging_path = staging.get("path")
        if not isinstance(staging_path, str) or not staging_path.startswith("/var/tmp/quicklearn-wechat-"):
            raise PublishError("remote gateway returned an invalid staging path")

        cover = bundle["cover_path"]
        remote_cover = f"{staging_path}/cover{cover.suffix.lower()}"
        upload_remote_file(args.host, args.remote_bin, cover, remote_cover)
        remote_images: list[dict[str, str]] = []
        for item in bundle["body_images"]:
            local_path = item["path"]
            remote_path = f"{staging_path}/{item['id']}{local_path.suffix.lower()}"
            upload_remote_file(args.host, args.remote_bin, local_path, remote_path)
            remote_images.append({"id": item["id"], "path": remote_path, "alt": item["alt"]})

        remote_payload = {
            "title": bundle["title"],
            "author": bundle["author"],
            "digest": bundle["digest"],
            "content": bundle["content"],
            "content_source_url": bundle["content_source_url"],
            "cover_path": remote_cover,
            "body_images": remote_images,
            "need_open_comment": bundle["need_open_comment"],
            "only_fans_can_comment": bundle["only_fans_can_comment"],
        }
        result = run_remote(
            args.host,
            args.remote_bin,
            ["create-draft", "--input", "-"],
            input_text=json.dumps(remote_payload, ensure_ascii=False),
        )
        result["transport"] = "ssh"
        result["host"] = args.host
        return result
    finally:
        if staging_path:
            try:
                run_remote(args.host, args.remote_bin, ["stage-clean", "--path", staging_path])
            except PublishError as exc:
                print(
                    json.dumps(
                        {"warning": "remote staging cleanup failed", "detail": str(exc)},
                        ensure_ascii=False,
                    ),
                    file=sys.stderr,
                )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Publish a 快学AI draft through SSH")
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--remote-bin", default=DEFAULT_REMOTE_BIN)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("doctor")
    publish_parser = commands.add_parser("publish")
    publish_parser.add_argument("manifest", type=Path)
    publish_parser.add_argument("--dry-run", action="store_true")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        if args.command == "doctor":
            result = run_remote(args.host, args.remote_bin, ["doctor"])
        else:
            result = publish(args)
        print(json.dumps(result, ensure_ascii=False, sort_keys=True))
        return 0
    except (PublishError, OSError, UnicodeError, subprocess.TimeoutExpired) as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
