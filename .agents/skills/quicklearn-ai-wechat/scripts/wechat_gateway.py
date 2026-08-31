#!/usr/bin/env python3
"""Server-side WeChat draft gateway invoked through SSH.

The program never prints credentials or access tokens and intentionally exposes
no publish, mass-send, draft-delete, or material-delete command.
"""

from __future__ import annotations

import argparse
import getpass
import hashlib
import json
import mimetypes
import os
import re
import secrets
import shutil
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


API_BASE = "https://api.weixin.qq.com"
DEFAULT_CONFIG = Path("/etc/quicklearn-wechat/credentials.json")
DEFAULT_STATE_DIR = Path("/var/lib/quicklearn-wechat")
STAGING_ROOT = Path("/var/tmp")
STAGING_PREFIX = "quicklearn-wechat-"
IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".gif", ".bmp"}
BODY_IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png"}
IMAGE_PLACEHOLDER = re.compile(r"\{\{IMAGE:([a-z0-9][a-z0-9-]{2,63})\}\}")
UNSAFE_HTML = re.compile(
    r"<(?:script|style|iframe|object|embed|form)\b|\son[a-z]+\s*=|javascript\s*:",
    re.IGNORECASE,
)
EXTERNAL_IMAGE = re.compile(
    r"<img\b[^>]*\bsrc\s*=\s*(['\"])(https?://(?!mmbiz\.qpic\.cn/|mmbiz\.qlogo\.cn/)[^'\"]+)\1",
    re.IGNORECASE,
)


class GatewayError(RuntimeError):
    """Safe-to-display gateway error."""


class WeChatError(GatewayError):
    def __init__(self, errcode: int, errmsg: str) -> None:
        self.errcode = errcode
        self.errmsg = errmsg
        super().__init__(f"WeChat API error {errcode}: {errmsg}")


def emit(value: dict[str, Any], *, stream: Any = sys.stdout) -> None:
    print(json.dumps(value, ensure_ascii=False, sort_keys=True), file=stream)


def load_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise GatewayError(f"cannot read JSON file {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise GatewayError(f"JSON file must contain an object: {path}")
    return value


def write_private_json(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
            json.dump(value, handle, ensure_ascii=False, sort_keys=True)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()


def load_config(path: Path) -> dict[str, str]:
    value = load_json(path)
    required = ("account_name", "appid", "appsecret")
    for key in required:
        if not isinstance(value.get(key), str) or not value[key].strip():
            raise GatewayError(f"credential configuration is missing {key}")
    return {key: value[key].strip() for key in required}


def response_json(request: urllib.request.Request, timeout: int = 30) -> dict[str, Any]:
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            data = response.read()
    except urllib.error.HTTPError as exc:
        raise GatewayError(f"WeChat HTTP error {exc.code}") from exc
    except urllib.error.URLError as exc:
        raise GatewayError(f"WeChat network error: {exc.reason}") from exc
    try:
        value = json.loads(data.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise GatewayError("WeChat returned a non-JSON response") from exc
    if not isinstance(value, dict):
        raise GatewayError("WeChat returned an unexpected JSON response")
    return value


def ensure_wechat_success(value: dict[str, Any]) -> dict[str, Any]:
    errcode = value.get("errcode")
    if isinstance(errcode, int) and errcode != 0:
        raise WeChatError(errcode, str(value.get("errmsg", "unknown error")))
    return value


class WeChatClient:
    def __init__(self, config_path: Path, state_dir: Path) -> None:
        self.config = load_config(config_path)
        self.state_dir = state_dir
        self.state_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
        os.chmod(self.state_dir, 0o700)

    @property
    def account_name(self) -> str:
        return self.config["account_name"]

    def access_token(self, *, force_refresh: bool = False) -> tuple[str, int]:
        cache_path = self.state_dir / "access-token.json"
        now = int(time.time())
        if not force_refresh and cache_path.exists():
            cache = load_json(cache_path)
            token = cache.get("access_token")
            expires_at = cache.get("expires_at")
            if isinstance(token, str) and isinstance(expires_at, int) and expires_at > now + 300:
                return token, expires_at - now

        payload = json.dumps(
            {
                "grant_type": "client_credential",
                "appid": self.config["appid"],
                "secret": self.config["appsecret"],
                "force_refresh": force_refresh,
            },
            ensure_ascii=False,
        ).encode("utf-8")
        request = urllib.request.Request(
            f"{API_BASE}/cgi-bin/stable_token",
            data=payload,
            headers={"Content-Type": "application/json; charset=utf-8"},
            method="POST",
        )
        value = ensure_wechat_success(response_json(request))
        token = value.get("access_token")
        expires_in = value.get("expires_in")
        if not isinstance(token, str) or not isinstance(expires_in, int):
            raise GatewayError("WeChat token response is missing required fields")
        write_private_json(
            cache_path,
            {"access_token": token, "expires_at": now + expires_in},
        )
        return token, expires_in

    def get_json(self, endpoint: str, token: str) -> dict[str, Any]:
        url = f"{API_BASE}{endpoint}?{urllib.parse.urlencode({'access_token': token})}"
        return ensure_wechat_success(response_json(urllib.request.Request(url, method="GET")))

    def post_json(self, endpoint: str, token: str, payload: dict[str, Any]) -> dict[str, Any]:
        url = f"{API_BASE}{endpoint}?{urllib.parse.urlencode({'access_token': token})}"
        data = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        request = urllib.request.Request(
            url,
            data=data,
            headers={"Content-Type": "application/json; charset=utf-8"},
            method="POST",
        )
        return ensure_wechat_success(response_json(request))

    def upload_file(self, endpoint: str, token: str, path: Path, *, query: dict[str, str]) -> dict[str, Any]:
        boundary = f"----quicklearn{secrets.token_hex(16)}"
        mime_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
        body = bytearray()
        body.extend(f"--{boundary}\r\n".encode())
        body.extend(
            (
                f'Content-Disposition: form-data; name="media"; filename="{path.name}"\r\n'
                f"Content-Type: {mime_type}\r\n\r\n"
            ).encode("utf-8")
        )
        body.extend(path.read_bytes())
        body.extend(f"\r\n--{boundary}--\r\n".encode())
        parameters = {"access_token": token, **query}
        url = f"{API_BASE}{endpoint}?{urllib.parse.urlencode(parameters)}"
        request = urllib.request.Request(
            url,
            data=bytes(body),
            headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
            method="POST",
        )
        return ensure_wechat_success(response_json(request, timeout=60))

    def upload_cover(self, token: str, path: Path) -> str:
        validate_image(path, BODY_IMAGE_EXTENSIONS | {".gif", ".bmp"}, 10 * 1024 * 1024)
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        cache_path = self.state_dir / "cover-cache.json"
        cache = load_json(cache_path) if cache_path.exists() else {}
        cached = cache.get(digest)
        if isinstance(cached, str) and cached:
            return cached
        value = self.upload_file(
            "/cgi-bin/material/add_material",
            token,
            path,
            query={"type": "image"},
        )
        media_id = value.get("media_id")
        if not isinstance(media_id, str) or not media_id:
            raise GatewayError("cover upload response is missing media_id")
        cache[digest] = media_id
        write_private_json(cache_path, cache)
        return media_id

    def upload_body_image(self, token: str, path: Path) -> str:
        validate_image(path, BODY_IMAGE_EXTENSIONS, 1024 * 1024)
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        cache_path = self.state_dir / "body-image-cache.json"
        cache = load_json(cache_path) if cache_path.exists() else {}
        cached = cache.get(digest)
        if isinstance(cached, str) and cached:
            return cached
        value = self.upload_file("/cgi-bin/media/uploadimg", token, path, query={})
        url = value.get("url")
        if not isinstance(url, str) or not url.startswith("https://"):
            raise GatewayError("body image upload response is missing a secure URL")
        cache[digest] = url
        write_private_json(cache_path, cache)
        return url


def validate_image(path: Path, extensions: set[str], max_bytes: int) -> None:
    resolved = path.resolve(strict=True)
    if not resolved.is_file():
        raise GatewayError(f"image is not a regular file: {path}")
    if resolved.suffix.lower() not in extensions:
        raise GatewayError(f"unsupported image type: {resolved.suffix.lower()}")
    size = resolved.stat().st_size
    if size <= 0 or size > max_bytes:
        raise GatewayError(f"image size {size} is outside the allowed range")


def require_text(payload: dict[str, Any], key: str, maximum: int) -> str:
    value = payload.get(key)
    if not isinstance(value, str) or not value.strip():
        raise GatewayError(f"{key} must be a non-empty string")
    value = value.strip()
    if len(value) > maximum:
        raise GatewayError(f"{key} exceeds {maximum} characters")
    return value


def validate_staging_path(raw: str) -> Path:
    path = Path(raw).resolve(strict=True)
    if path.parent != STAGING_ROOT or not path.name.startswith(STAGING_PREFIX):
        raise GatewayError("refusing path outside the managed staging directory")
    if not path.is_dir():
        raise GatewayError("staging path is not a directory")
    return path


def create_draft(client: WeChatClient, payload: dict[str, Any]) -> dict[str, Any]:
    title = require_text(payload, "title", 32)
    author = require_text(payload, "author", 16)
    digest = payload.get("digest", "")
    if not isinstance(digest, str) or len(digest) > 120:
        raise GatewayError("digest must be a string of at most 120 characters")
    content = require_text(payload, "content", 20000)
    if len(content.encode("utf-8")) >= 1024 * 1024:
        raise GatewayError("content must be smaller than 1 MiB")
    if UNSAFE_HTML.search(content):
        raise GatewayError("content contains unsafe HTML")
    if EXTERNAL_IMAGE.search(content):
        raise GatewayError("content contains an external image URL")

    source_url = payload.get("content_source_url", "")
    if not isinstance(source_url, str) or len(source_url.encode("utf-8")) > 1024:
        raise GatewayError("content_source_url is invalid or too long")
    if source_url and not source_url.startswith(("https://", "http://")):
        raise GatewayError("content_source_url must be an HTTP(S) URL")

    token, _ = client.access_token()
    cover_path = payload.get("cover_path")
    if not isinstance(cover_path, str) or not cover_path:
        raise GatewayError("cover_path is required")
    thumb_media_id = client.upload_cover(token, Path(cover_path))

    body_images = payload.get("body_images", [])
    if not isinstance(body_images, list):
        raise GatewayError("body_images must be an array")
    seen_ids: set[str] = set()
    for item in body_images:
        if not isinstance(item, dict):
            raise GatewayError("body image entry must be an object")
        image_id = item.get("id")
        image_path = item.get("path")
        alt = item.get("alt", "")
        if not isinstance(image_id, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{2,63}", image_id):
            raise GatewayError("body image id is invalid")
        if image_id in seen_ids:
            raise GatewayError(f"duplicate body image id: {image_id}")
        seen_ids.add(image_id)
        placeholder = f"{{{{IMAGE:{image_id}}}}}"
        if content.count(placeholder) != 1:
            raise GatewayError(f"body image placeholder must appear exactly once: {image_id}")
        if not isinstance(image_path, str) or not image_path:
            raise GatewayError(f"body image path is missing: {image_id}")
        if not isinstance(alt, str) or len(alt) > 160:
            raise GatewayError(f"body image alt text is invalid: {image_id}")
        url = client.upload_body_image(token, Path(image_path))
        safe_alt = (
            alt.replace("&", "&amp;")
            .replace('"', "&quot;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
        )
        image_html = (
            f'<p style="margin:24px 0;text-align:center;">'
            f'<img src="{url}" alt="{safe_alt}" '
            f'style="max-width:100%;height:auto;display:block;margin:0 auto;"/></p>'
        )
        content = content.replace(placeholder, image_html)

    unresolved = IMAGE_PLACEHOLDER.findall(content)
    if unresolved:
        raise GatewayError(f"unresolved body image placeholders: {sorted(set(unresolved))}")

    article = {
        "article_type": "news",
        "title": title,
        "author": author,
        "digest": digest,
        "content": content,
        "content_source_url": source_url,
        "thumb_media_id": thumb_media_id,
        "need_open_comment": 1 if payload.get("need_open_comment") else 0,
        "only_fans_can_comment": 1 if payload.get("only_fans_can_comment") else 0,
    }
    created = client.post_json("/cgi-bin/draft/add", token, {"articles": [article]})
    media_id = created.get("media_id")
    if not isinstance(media_id, str) or not media_id:
        raise GatewayError("draft response is missing media_id")

    verified = client.post_json("/cgi-bin/draft/get", token, {"media_id": media_id})
    news_items = verified.get("news_item")
    if not isinstance(news_items, list) or len(news_items) != 1:
        raise GatewayError("draft verification returned an unexpected article count")
    verified_title = news_items[0].get("title") if isinstance(news_items[0], dict) else None
    if verified_title != title:
        raise GatewayError("draft verification title does not match")
    return {
        "ok": True,
        "verified": True,
        "account": client.account_name,
        "title": title,
        "media_id": media_id,
        "article_count": 1,
        "body_image_count": len(body_images),
        "cover_cached": True,
    }


def parse_payload(path: str) -> dict[str, Any]:
    if path == "-":
        try:
            value = json.load(sys.stdin)
        except json.JSONDecodeError as exc:
            raise GatewayError(f"invalid JSON input: {exc}") from exc
        if not isinstance(value, dict):
            raise GatewayError("input JSON must be an object")
        return value
    return load_json(Path(path))


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="QuickLearn AI WeChat draft gateway")
    parser.add_argument("--config", type=Path, default=DEFAULT_CONFIG)
    parser.add_argument("--state-dir", type=Path, default=DEFAULT_STATE_DIR)
    commands = parser.add_subparsers(dest="command", required=True)
    configure = commands.add_parser("configure")
    configure.add_argument("--account-name", default="快学AI")
    commands.add_parser("doctor")
    commands.add_parser("stage-create")
    put = commands.add_parser("stage-put")
    put.add_argument("--path", required=True)
    clean = commands.add_parser("stage-clean")
    clean.add_argument("--path", required=True)
    create = commands.add_parser("create-draft")
    create.add_argument("--input", default="-")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        if args.command == "configure":
            appid = input("AppID: ").strip()
            appsecret = getpass.getpass("AppSecret: ").strip()
            if not re.fullmatch(r"wx[a-fA-F0-9]{16}", appid):
                raise GatewayError("AppID format is invalid")
            if not appsecret:
                raise GatewayError("AppSecret cannot be empty")
            write_private_json(
                args.config,
                {
                    "account_name": args.account_name,
                    "appid": appid,
                    "appsecret": appsecret,
                },
            )
            emit({"ok": True, "configured": True, "account": args.account_name})
            return 0
        if args.command == "stage-create":
            path = Path(tempfile.mkdtemp(prefix=STAGING_PREFIX, dir=STAGING_ROOT))
            os.chmod(path, 0o700)
            emit({"ok": True, "path": str(path)})
            return 0
        if args.command == "stage-put":
            target = Path(args.path)
            parent = validate_staging_path(str(target.parent))
            if target.parent.resolve() != parent or not re.fullmatch(r"[a-z0-9][a-z0-9._-]{1,127}", target.name):
                raise GatewayError("invalid staging file path")
            data = sys.stdin.buffer.read(10 * 1024 * 1024 + 1)
            if not data or len(data) > 10 * 1024 * 1024:
                raise GatewayError("staging file is empty or exceeds 10 MiB")
            descriptor = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(descriptor, "wb") as handle:
                handle.write(data)
                handle.flush()
                os.fsync(handle.fileno())
            emit({"ok": True, "stored": True, "bytes": len(data)})
            return 0
        if args.command == "stage-clean":
            path = validate_staging_path(args.path)
            shutil.rmtree(path)
            emit({"ok": True, "cleaned": True})
            return 0

        client = WeChatClient(args.config, args.state_dir)
        if args.command == "doctor":
            token, expires_in = client.access_token()
            count = client.get_json("/cgi-bin/draft/count", token)
            emit(
                {
                    "ok": True,
                    "account": client.account_name,
                    "token_valid": True,
                    "token_expires_in": expires_in,
                    "draft_count": count.get("total_count"),
                }
            )
            return 0
        if args.command == "create-draft":
            emit(create_draft(client, parse_payload(args.input)))
            return 0
        raise GatewayError("unsupported command")
    except WeChatError as exc:
        emit(
            {"ok": False, "error": "wechat_api_error", "errcode": exc.errcode, "errmsg": exc.errmsg},
            stream=sys.stderr,
        )
        return 2
    except (GatewayError, OSError) as exc:
        emit({"ok": False, "error": str(exc)}, stream=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
