import { chromium } from "playwright";

const BASE = "http://127.0.0.1:8088";

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const errors = [];
  page.on("pageerror", (err) => errors.push(err.message));

  console.log("=== Testing / (homepage) ===");
  try {
    await page.goto(BASE + "/", { waitUntil: "networkidle", timeout: 15000 });
    const title = await page.title();
    const bodyText = await page.locator("body").innerText({ timeout: 5000 }).catch(() => "(empty)");
    console.log("  title:", title);
    console.log("  body length:", bodyText.length, "chars");
    console.log("  first 200 chars:", bodyText.substring(0, 200));
    if (errors.length) {
      console.log("  ERRORS:");
      errors.forEach(e => console.log("    ", e));
      errors.length = 0;
    } else {
      console.log("  No errors");
    }
  } catch (e) {
    console.log("  FAILED:", e.message);
  }

  console.log("\n=== Testing /about ===");
  errors.length = 0;
  try {
    await page.goto(BASE + "/about", { waitUntil: "networkidle", timeout: 15000 });
    const bodyText = await page.locator("body").innerText({ timeout: 5000 }).catch(() => "(empty)");
    console.log("  body length:", bodyText.length, "chars");
    console.log("  first 200 chars:", bodyText.substring(0, 200));
    if (errors.length) {
      console.log("  ERRORS:");
      errors.forEach(e => console.log("    ", e));
      errors.length = 0;
    } else {
      console.log("  No errors");
    }
  } catch (e) {
    console.log("  FAILED:", e.message);
  }

  console.log("\n=== Testing /contact ===");
  errors.length = 0;
  try {
    await page.goto(BASE + "/contact", { waitUntil: "networkidle", timeout: 15000 });
    const bodyText = await page.locator("body").innerText({ timeout: 5000 }).catch(() => "(empty)");
    console.log("  body length:", bodyText.length, "chars");
    if (errors.length) {
      console.log("  ERRORS:");
      errors.forEach(e => console.log("    ", e));
      errors.length = 0;
    } else {
      console.log("  No errors");
    }
  } catch (e) {
    console.log("  FAILED:", e.message);
  }

  await browser.close();
}

main().catch(console.error);
