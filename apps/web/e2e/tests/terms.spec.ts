import { execFileSync } from "node:child_process";
import path from "node:path";
import { expect, test } from "@playwright/test";

// PLAN Tahap 7: no wording outside the glossary in anything a person reads (DESIGN.md §7). The checker looks at JSX text,
// labels, placeholders, toasts and error messages, not at identifiers or API paths.
test("no UI text uses a term outside the glossary", () => {
  const script = path.resolve(process.cwd(), "scripts/check-ui-terms.mjs");
  let output = "";
  try {
    output = execFileSync("node", [script], { encoding: "utf8" });
  } catch (err) {
    output = (err as { stdout?: string }).stdout ?? String(err);
    expect(output, "wording outside the glossary").toBe("");
  }
  expect(output).toContain("no wording outside the glossary");
});
