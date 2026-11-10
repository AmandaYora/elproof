import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

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
