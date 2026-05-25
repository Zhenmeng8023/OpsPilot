import { defineConfig } from "@playwright/test";

const baseURL = process.env.SMOKE_BASE_URL ?? "http://127.0.0.1:5173";

export default defineConfig({
  testDir: "./tests",
  outputDir: "../artifacts/ui-smoke/latest/test-results",
  timeout: 90_000,
  expect: {
    timeout: 10_000
  },
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI
    ? [["github"], ["json", { outputFile: "../artifacts/ui-smoke/latest/summary.json" }], ["html", { open: "never", outputFolder: "../artifacts/ui-smoke/latest/html-report" }]]
    : [["list"], ["json", { outputFile: "../artifacts/ui-smoke/latest/summary.json" }], ["html", { open: "never", outputFolder: "../artifacts/ui-smoke/latest/html-report" }]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "on",
    video: "retain-on-failure"
  },
  projects: [
    {
      name: "desktop",
      use: { viewport: { width: 1440, height: 900 } }
    },
    {
      name: "narrow",
      use: { viewport: { width: 390, height: 844 } }
    }
  ]
});
