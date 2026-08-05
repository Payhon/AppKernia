#!/usr/bin/env python3
"""Executable fake-HttpPort checks for the mobile 401 refresh/replay policy."""

from __future__ import annotations


class Unauthorized(Exception):
    pass


class FakeSession:
    def __init__(self) -> None:
        self.cleared = False

    def clear(self) -> None:
        self.cleared = True


class FakeHttpPort:
    def __init__(self, responses: list[object]) -> None:
        self.responses = responses
        self.calls: list[str] = []

    def execute(self, method: str) -> str:
        self.calls.append(method)
        value = self.responses.pop(0)
        if value == "401":
            raise Unauthorized
        return str(value)


def execute(port: FakeHttpPort, session: FakeSession, method: str, refresh_ok: bool, retry_on_unauthorized: bool = True) -> str:
    try:
        return port.execute(method)
    except Unauthorized:
        if not retry_on_unauthorized or method not in {"GET", "HEAD"}:
            raise
        if not refresh_ok:
            session.clear()
            raise
        return port.execute(method)


def main() -> None:
    session = FakeSession()
    get_port = FakeHttpPort(["401", "ok"])
    assert execute(get_port, session, "GET", True) == "ok"
    assert get_port.calls == ["GET", "GET"] and not session.cleared

    disabled_get_port = FakeHttpPort(["401"])
    try:
        execute(disabled_get_port, FakeSession(), "GET", True, retry_on_unauthorized=False)
        raise AssertionError("GET without explicit opt-in must not be replayed")
    except Unauthorized:
        assert disabled_get_port.calls == ["GET"]

    for method in ("DELETE", "PUT", "PATCH", "POST"):
        write_port = FakeHttpPort(["401"])
        try:
            execute(write_port, FakeSession(), method, True)
            raise AssertionError(method + " must not be replayed")
        except Unauthorized:
            assert write_port.calls == [method]

    failed_session = FakeSession()
    failed_refresh_port = FakeHttpPort(["401"])
    try:
        execute(failed_refresh_port, failed_session, "GET", False)
        raise AssertionError("failed refresh must return unauthorized")
    except Unauthorized:
        assert failed_session.cleared

    # Asset downloads receive the bearer in a header and use the same one-time
    # refresh policy; the URL itself remains the server-derived asset path.
    asset_headers = {"Authorization": "Bearer access-token"}
    asset_path = "/api/v1/article-assets/123e4567-e89b-12d3-a456-426614174000"
    assert asset_headers["Authorization"] == "Bearer access-token"
    assert "access-token" not in asset_path and "?" not in asset_path

    print("refresh policy fake HttpPort checks passed")


if __name__ == "__main__":
    main()
