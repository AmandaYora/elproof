import { expect, state, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";

const API = (path: string) => `${apiURL()}/api/v1${path}`;

// T-20261110-11 (ADR-0039): a WO keeps its logo, but no longer has a colour of its own. This test WRITES a logo, so it runs in the
// "mutating" project, after every screenshot.
test("a WO's logo shows in its console, and the per-WO colour is gone from the API and Profil Usaha", async ({ browser }, info) => {
  test.skip(!state().storage, "needs object storage (MinIO) for the logo");
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/profil-usaha");
  await expect(page.getByRole("heading", { name: "Profil Usaha" })).toBeVisible();
  await expect(page.getByText("Warna merek")).toHaveCount(0);
  await expect(page.getByRole("radiogroup")).toHaveCount(0);

  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string });
  const auth = { Authorization: `Bearer ${session.accessToken}` };

  // The colour endpoint and field no longer exist.
  const branding = await (await page.request.get(API("/tenants/me/branding"), { headers: auth })).json();
  expect(Object.keys(branding.data).sort()).toEqual(["businessName", "hasLogo"]);
  const patch = await page.request.patch(API("/tenants/me/brand-color"), { headers: auth, data: { preset: "gold" } });
  expect(patch.status()).toBe(404);

  // A logo drawn in the browser (plum mark, read from the theme), uploaded the way Profil Usaha does.
  const base64Data = await page.evaluate(() => {
    const c = document.createElement("canvas");
    c.width = 240;
    c.height = 64;
    const g = c.getContext("2d")!;
    const plum = getComputedStyle(document.documentElement).getPropertyValue("--primary").trim();
    g.fillStyle = plum;
    g.beginPath();
    g.arc(32, 32, 28, 0, Math.PI * 2);
    g.fill();
    g.font = "bold 30px sans-serif";
    g.fillText("Berbayar", 72, 43);
    return c.toDataURL("image/png").split(",")[1];
  });
  const put = await page.request.put(API("/tenants/me/logo"), { headers: auth, data: { fileName: "logo.png", mimeType: "image/png", base64Data } });
  expect(put.status()).toBe(200);

  await visit(page, "/dashboard");
  const logo = page.getByRole("img", { name: branding.data.businessName, exact: true }).first();
  await expect(logo).toBeVisible();
  expect(await logo.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
  await page.screenshot({ path: info.outputPath("wo-logo-desktop.png") });
  await info.attach("wo-logo-desktop", { path: info.outputPath("wo-logo-desktop.png"), contentType: "image/png" });

  await visit(page, "/profil-usaha");
  await expect(page.getByRole("img", { name: /^Logo / })).toBeVisible();
  await close();
});
