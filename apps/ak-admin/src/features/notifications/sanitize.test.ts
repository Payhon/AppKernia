// @vitest-environment jsdom

import { describe, expect, it } from "vitest";
import { sanitizeNotificationHtml } from "./sanitize";

describe("sanitizeNotificationHtml", () => {
  it("matches the server notification allowlist boundary", () => {
    const value = sanitizeNotificationHtml(
      `<p onclick="bad()">Hello<script>alert(1)</script><a href="javascript:bad()">bad</a><a href="https://example.com" target="_blank">safe</a></p>`,
    );
    expect(value).toBe(
      `<p>Hello<a>bad</a><a href="https://example.com">safe</a></p>`,
    );
  });
});
