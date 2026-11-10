import type { Page } from "@playwright/test";
import { expect, prepareContext, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { AUTH, apiURL } from "../lib/config";

const API = (path: string) => `${apiURL()}/api/v1${path}`;

async function newProject(page: Page, name: string) {
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string; principalId: string });
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const made = await page.request.post(API("/projects"), {
    headers: auth,
    data: {
      name, brideName: "Lina Kusuma", groomName: "Reza Mahendra", eventDate: "2027-07-01", venue: "Gedung Uji", prepStartDate: "2026-11-09",
      packageName: "Paket Uji", contractValue: 10_000_000, status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  return { id: ((await made.json()).data as { id: number }).id, auth };
}

// PLAN Tahap 8: animations respect prefers-reduced-motion; two sessions never overwrite each other silently (ADR-0036).
// These tests WRITE data, so they run in the "mutating" project.

test.describe("reduced motion", () => {
  for (const route of ["/dashboard", "/projects/1/milestone"]) {
    test(`${route} has no running transition or animation longer than a frame`, async ({ browser }, info) => {
      const context = await browser.newContext({ ...info.project.use, storageState: AUTH.owner, reducedMotion: "reduce" });
      const page = await context.newPage();
      await prepareContext(page);
      await visit(page, route);
      const offenders = await page.evaluate(() => {
        const longest = (value: string) => Math.max(0, ...value.split(",").map((v) => parseFloat(v) * (v.trim().endsWith("ms") ? 0.001 : 1)));
        const bad: string[] = [];
        for (const el of Array.from(document.querySelectorAll("*"))) {
          const cs = getComputedStyle(el);
          if (longest(cs.transitionDuration) > 0.011 || longest(cs.animationDuration) > 0.011) bad.push(`${el.tagName.toLowerCase()}.${String(el.className).slice(0, 60)}`);
        }
        return bad.slice(0, 5);
      });
      expect(offenders).toEqual([]);
      await context.close();
    });
  }
});

test.describe("two sessions on one item", () => {
  test("a project edit made over someone else's change is refused, nothing is overwritten, and Muat ulang shows theirs", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/dashboard");
    const { id, auth } = await newProject(page, "Uji Konflik");
    await visit(page, `/projects/${id}/ringkasan`);

    // Session A opens the form...
    await page.getByRole("button", { name: "Ubah proyek" }).click();
    const dialog = page.getByRole("dialog", { name: "Ubah proyek" });
    await expect(dialog).toBeVisible();

    // ...while session B (the API, as another person) saves a change to the same project.
    const theirs = await page.request.patch(API(`/projects/${id}`), { headers: auth, data: { name: "Uji Konflik, ubahan B" } });
    expect(theirs.status()).toBe(200);

    await dialog.getByLabel("Deskripsi / catatan proyek").fill("Catatan dari A");
    await dialog.getByRole("button", { name: "Simpan perubahan" }).click();

    const refused = page.getByRole("alertdialog", { name: "Data diubah orang lain" });
    await expect(refused).toBeVisible();
    // Nothing of A's was written, and B's change is intact.
    const now = (await (await page.request.get(API(`/projects/${id}`), { headers: auth })).json()).data as { name: string; description: string };
    expect(now.name).toBe("Uji Konflik, ubahan B");
    expect(now.description).toBe("");

    await refused.getByRole("button", { name: "Muat ulang" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Uji Konflik, ubahan B" })).toBeVisible();
    await expect(page.getByRole("dialog", { name: "Ubah proyek" })).toBeHidden(); // the stale form is gone with the old page
    await close();
  });

  test("changing a step's status from the list over someone else's edit of that step is refused", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/dashboard");
    const { id, auth } = await newProject(page, "Uji Konflik Tahapan");
    await visit(page, `/projects/${id}/milestone`);
    const firstRow = page.getByRole("row").filter({ hasText: /^1\./ }).first();
    await expect(firstRow).toBeVisible();

    const listed = (await (await page.request.get(API(`/projects/${id}/milestones`), { headers: auth })).json()).data as Array<{ id: number; targetDate: string }>;
    const bumped = await page.request.patch(API(`/projects/${id}/milestones/${listed[0].id}`), { headers: auth, data: { targetDate: "2027-06-01" } });
    expect(bumped.status()).toBe(200);

    const status = firstRow.getByRole("button", { name: /Belum mulai/ });
    await status.scrollIntoViewIfNeeded();
    await page.waitForTimeout(200);
    await status.click();
    await page.getByRole("option", { name: "Selesai" }).click();

    await expect(page.getByRole("alertdialog", { name: "Data diubah orang lain" })).toBeVisible();
    const after = (await (await page.request.get(API(`/projects/${id}/milestones`), { headers: auth })).json()).data as Array<{ status: string; targetDate: string }>;
    expect(after[0].status).toBe("Not Started"); // A's change did not land
    expect(after[0].targetDate).toBe("2027-06-01"); // B's did
    await close();
  });

  test("the API gives every edited item a version and refuses an old one; without it the write goes through", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/dashboard");
    const { id, auth } = await newProject(page, "Uji Versi");
    const read = async () => (await (await page.request.get(API(`/projects/${id}`), { headers: auth })).json()).data as { updatedAt: string };
    const v1 = (await read()).updatedAt;
    expect(v1).toMatch(/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{6}Z$/);

    const fresh = await page.request.patch(API(`/projects/${id}`), { headers: { ...auth, "If-Match": `"${v1}"` }, data: { description: "satu" } });
    expect(fresh.status()).toBe(200);
    const stale = await page.request.patch(API(`/projects/${id}`), { headers: { ...auth, "If-Match": `"${v1}"` }, data: { description: "dua" } });
    expect(stale.status()).toBe(409);
    const body = await stale.json();
    expect(body.errors._conflict).toEqual(["stale"]);
    expect(body.data.current.description).toBe("satu");
    // A caller that sends no version still works (transition).
    const blind = await page.request.patch(API(`/projects/${id}`), { headers: auth, data: { description: "tiga" } });
    expect(blind.status()).toBe(200);
    await close();
  });

  test("archiving and activating are explicit: saying it twice leaves it as said", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/dashboard");
    const { id, auth } = await newProject(page, "Uji Arsip");
    for (let i = 0; i < 2; i++) {
      const res = await page.request.patch(API(`/projects/${id}`), { headers: auth, data: { isArchived: true } });
      expect(res.status()).toBe(200);
      expect(((await res.json()).data as { isArchived: boolean }).isArchived).toBe(true);
    }
    const back = await page.request.patch(API(`/projects/${id}`), { headers: auth, data: { isArchived: false } });
    expect(((await back.json()).data as { isArchived: boolean }).isArchived).toBe(false);
    await close();
  });

  test("one person's own sequence of edits never collides with itself", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/dashboard");
    const { id, auth } = await newProject(page, "Uji Urutan");
    await visit(page, `/projects/${id}/ringkasan`);
    const refused = page.getByRole("alertdialog", { name: "Data diubah orang lain" });

    // Edit, then edit again without reloading: the second save carries the version the first one returned.
    for (const note of ["Catatan satu", "Catatan dua"]) {
      await page.getByRole("button", { name: "Ubah proyek" }).click();
      const dialog = page.getByRole("dialog", { name: "Ubah proyek" });
      await dialog.getByLabel("Deskripsi / catatan proyek").fill(note);
      await dialog.getByRole("button", { name: "Simpan perubahan" }).click();
      await expect(dialog).toBeHidden();
      await expect(refused).toBeHidden();
    }

    // A status change on the project (its own call, which also moves the row's version), then an edit.
    await page.getByRole("button", { name: "Mulai persiapan" }).click();
    await expect(page.getByRole("button", { name: "Tandai siap" })).toBeVisible(); // Draft -> Persiapan: the next step is offered
    await page.getByRole("button", { name: "Ubah proyek" }).click();
    const dialog = page.getByRole("dialog", { name: "Ubah proyek" });
    await dialog.getByLabel("Deskripsi / catatan proyek").fill("Catatan tiga");
    await dialog.getByRole("button", { name: "Simpan perubahan" }).click();
    await expect(dialog).toBeHidden();
    await expect(refused).toBeHidden();
    const saved = (await (await page.request.get(API(`/projects/${id}`), { headers: auth })).json()).data as { description: string };
    expect(saved.description).toBe("Catatan tiga"); // every save landed
    await close();
  });

  test("the answer to a status change carries the version that is current, so the next edit can use it", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/dashboard");
    const { id, auth } = await newProject(page, "Uji Versi Status");
    const current = async () => ((await (await page.request.get(API(`/projects/${id}`), { headers: auth })).json()).data as { updatedAt: string }).updatedAt;
    const moved = await page.request.post(API(`/projects/${id}/transition`), { headers: auth, data: { action: "start_preparation" } });
    expect(moved.status()).toBe(200);
    const answered = ((await moved.json()).data as { updatedAt: string }).updatedAt;
    expect(answered).toBe(await current());
    expect(moved.headers()["etag"]).toBe(`"${answered}"`);
    // ...and an edit that sends it is accepted.
    const edit = await page.request.patch(API(`/projects/${id}`), { headers: { ...auth, "If-Match": `"${answered}"` }, data: { description: "setelah status" } });
    expect(edit.status()).toBe(200);
    const cancelled = await page.request.post(API(`/projects/${id}/cancel`), { headers: auth });
    expect(cancelled.status()).toBe(200);
    expect(((await cancelled.json()).data as { updatedAt: string }).updatedAt).toBe(await current());
    await close();
  });
});

