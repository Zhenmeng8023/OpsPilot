import { expect, test } from "@playwright/test";

const username = process.env.SMOKE_USERNAME ?? "admin";
const password = process.env.SMOKE_PASSWORD ?? "Admin@123456";

const routes = [
  "/dashboard",
  "/agents",
  "/metrics",
  "/incidents",
  "/webhooks",
  "/workflows",
  "/notifications",
  "/traces",
  "/security-review",
  "/audit-logs"
];

test("login and open production critical pages", async ({ page }) => {
  await page.goto("/login", { waitUntil: "domcontentloaded" });
  await expect(page.locator("form.login-card")).toBeVisible();

  await page.locator('input[autocomplete="username"]').fill(username);
  await page.locator('input[type="password"]').fill(password);

  await Promise.all([
    page.waitForURL("**/dashboard"),
    page.locator("form.login-card").press("Enter")
  ]);
  await expect(page.locator("main.page h1")).toBeVisible();

  for (const route of routes) {
    await page.goto(route, { waitUntil: "domcontentloaded" });
    await expect(page).toHaveURL(new RegExp(`${route.replace("/", "\\/")}$`));
    await expect(page.locator("main.page")).toBeVisible();
    await expect(page.locator("main.page h1")).toBeVisible();
  }
});
