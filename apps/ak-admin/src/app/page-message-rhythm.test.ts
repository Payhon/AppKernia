import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

const styles = readFileSync(new URL("../styles.css", import.meta.url), "utf8");

describe("Admin page message rhythm", () => {
  it("keeps a shared 16px gap between page alerts and following content", () => {
    expect(styles).toMatch(
      /\.ak-page-container\s+\.ant-alert\s*\+\s*\*\s*\{[^}]*margin-block-start:\s*16px;/s,
    );
  });

  it("does not apply the gap globally to modal and drawer alerts", () => {
    expect(styles).not.toMatch(/(^|\n)\.ant-alert\s*\+\s*\*/);
  });
});
