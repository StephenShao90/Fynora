#!/usr/bin/env node

import fs from "node:fs/promises";
import path from "node:path";
import { createRequire } from "node:module";

const require = createRequire(new URL("../apps/web/package.json", import.meta.url));
const { chromium } = require("playwright");

const baseURL = process.env.DEMO_BASE_URL || "http://localhost:3000";
const outputDir = path.resolve(process.cwd(), "docs/assets/demo-video");

async function wait(page, ms = 900) {
  await page.waitForTimeout(ms);
}

async function safeClick(page, label, options = {}) {
  const locator = page.getByRole("button", { name: label }).or(page.getByRole("link", { name: label })).first();
  if (await locator.count()) {
    await locator.click(options);
    await wait(page);
    return true;
  }
  return false;
}

async function openPage(page, route, heading) {
  await page.goto(`${baseURL}${route}`, { waitUntil: "networkidle" });
  if (heading) {
    await page.getByText(heading, { exact: false }).first().waitFor({ timeout: 8000 }).catch(() => {});
  }
  await wait(page, 1200);
}

await fs.rm(outputDir, { recursive: true, force: true });
await fs.mkdir(outputDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
  recordVideo: {
    dir: outputDir,
    size: { width: 1440, height: 1000 }
  }
});

const page = await context.newPage();
page.setDefaultTimeout(8000);

await page.goto(baseURL, { waitUntil: "networkidle" });
await wait(page, 1400);
await safeClick(page, "Open guided demo");
await wait(page, 1600);

await safeClick(page, "Continue");
await safeClick(page, "Start daily close");
await wait(page, 1200);

await openPage(page, "/dashboard", "Operations dashboard");
await safeClick(page, "Run full demo setup");
await safeClick(page, "Help");
await wait(page, 1500);

await openPage(page, "/imports", "Connect data");
await safeClick(page, "Help");
await safeClick(page, "Create sandbox test connection");
await safeClick(page, "Sync connected banks");
await wait(page, 1000);

await openPage(page, "/reconciliation", "Payout reconciliation");
await safeClick(page, "Help");
await safeClick(page, "Sync processor data");
await safeClick(page, "Sync bank deposits");
await safeClick(page, "Run reconciliation");
await safeClick(page, "Resolve break");
await wait(page, 1400);

await openPage(page, "/insights", "Cash forecast");
await safeClick(page, "Help");
await wait(page, 1400);

await openPage(page, "/transactions", "Transaction ledger");
await safeClick(page, "Help");
await wait(page, 1400);

await openPage(page, "/integrations", "Provider health");
await safeClick(page, "Help");
await wait(page, 1400);

await openPage(page, "/ops", "Control center");
await safeClick(page, "Help");
await wait(page, 1400);

await openPage(page, "/settings", "Team settings");
await safeClick(page, "Help");
await wait(page, 1800);

await page.close();
await context.close();
await browser.close();

const files = await fs.readdir(outputDir);
const video = files.find((file) => file.endsWith(".webm"));
if (!video) {
  throw new Error("Playwright did not produce a demo video");
}
const from = path.join(outputDir, video);
const to = path.resolve(process.cwd(), "docs/assets/clearflow-main-feature-demo.webm");
await fs.rename(from, to);
await fs.rm(outputDir, { recursive: true, force: true });
console.log(`[record-demo] wrote ${to}`);
