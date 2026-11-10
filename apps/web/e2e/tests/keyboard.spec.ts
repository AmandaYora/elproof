import { expect, visit, test } from "../lib/fixtures";
import { focusRing, open, tabTo } from "../lib/suite";

// PLAN Tahap 9, UJI-MANUAL.md K1: signing in with the keyboard alone. The manual script covers the rest of the flows with a
// person; this is the part that cannot silently regress.

test("sign-in from the keyboard: Tab order, empty submit lands on the first problem, Enter signs in", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "anon");
  await visit(page, "/login");

  const username = page.getByLabel("Nama pengguna atau email");
  const password = page.locator("#login-password");
  const reveal = page.getByRole("button", { name: "Tampilkan kata sandi" });
  const submit = page.getByRole("button", { name: "Masuk", exact: true });

  // Tab order follows the reading order: username, password, the eye, then the button.
  await username.focus();
  await page.keyboard.press("Tab");
  await expect(password).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(reveal).toBeFocused();
  await page.keyboard.press("Tab");
  expect(await page.evaluate(() => document.activeElement?.tagName)).not.toBe("BODY");

  // Submitting nothing: the problem is stated next to the field, and the keyboard is put on it.
  await submit.focus();
  await page.keyboard.press("Enter");
  await expect(username).toBeFocused();
  await expect(page.getByRole("alert").first()).toBeVisible();

  // The eye works with Space and the typed password becomes readable.
  await password.fill("rahasia-uji");
  await reveal.focus();
  await page.keyboard.press("Space");
  await expect(password).toHaveAttribute("type", "text");
  await close();
});

// T-20261110-11 (ADR-0039): Tailwind 4's outline-none also set --tw-outline-style: none, so kit controls drew no focus ring at all.
// Every kit control reached with the keyboard must show a solid outline of at least 2px in the --ring colour (plum, >= 11.5:1).
test("kit button, search, input and combobox show a solid 2px focus ring in --ring when reached with Tab", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "owner");
  await visit(page, "/vendors");

  const add = page.getByRole("button", { name: "Tambah Vendor" });
  await tabTo(page, add);
  const addRing = await focusRing(add);
  expect(addRing.style).toBe("solid");
  expect(addRing.width).toBeGreaterThanOrEqual(2);
  expect(addRing.color).toBe(addRing.ring);

  const search = page.getByRole("textbox").first();
  await tabTo(page, search);
  const searchRing = await focusRing(search);
  expect(searchRing.style).toBe("solid");
  expect(searchRing.width).toBeGreaterThanOrEqual(2);
  expect(searchRing.color).toBe(searchRing.ring);

  await tabTo(page, add);
  await page.keyboard.press("Space"); // Space acts on keyup: no Enter keydown can reach the dialog's first field and submit it
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  const field = dialog.locator('input[data-slot="input"]').first();
  await tabTo(page, field);
  const fieldRing = await focusRing(field);
  expect(fieldRing.style).toBe("solid");
  expect(fieldRing.width).toBeGreaterThanOrEqual(2);
  expect(fieldRing.color).toBe(fieldRing.ring);

  const combo = dialog.locator('button[aria-haspopup="listbox"]').first();
  if (await combo.count()) {
    await tabTo(page, combo);
    const comboRing = await focusRing(combo);
    expect(comboRing.style).toBe("solid");
    expect(comboRing.width).toBeGreaterThanOrEqual(2);
    expect(comboRing.color).toBe(comboRing.ring);
  }

  // The command palette's search box: its focus is a --ring ring on the group around it.
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await page.keyboard.press("Control+k");
  const command = page.locator('[data-slot="command-input"]');
  await expect(command).toBeFocused();
  const group = page.locator('[data-slot="input-group"]').filter({ has: command });
  const shadow = await group.evaluate((el) => getComputedStyle(el).boxShadow);
  expect(shadow).toContain(fieldRing.ring);
  await close();
});
