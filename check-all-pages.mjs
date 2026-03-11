import { chromium } from "playwright";

const BASE = "http://127.0.0.1:8088";
const ROUTES = ["/", "/about", "/advantages", "/core-services", "/cases", "/experts", "/contact", "/blog", "/admin/theme"];

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  let allPassed = true;

  for (const route of ROUTES) {
    const errors = [];
    page.on("pageerror", (err) => errors.push(err.message));

    console.log(`=== Testing ${route} ===`);
    try {
      await page.goto(BASE + route, { waitUntil: "networkidle", timeout: 15000 });
      const bodyText = await page.locator("body").innerText({ timeout: 5000 }).catch(() => "(empty)");
      console.log(`  body length: ${bodyText.length} chars`);
      if (errors.length) {
        console.log("  ERRORS:");
        errors.forEach(e => console.log("    ", e));
        allPassed = false;
      } else {
        console.log("  OK");
      }
    } catch (e) {
      console.log("  FAILED:", e.message);
      allPassed = false;
    }

    page.removeAllListeners("pageerror");
  }

  await browser.close();
  console.log("\n" + (allPassed ? "ALL PAGES PASSED" : "SOME PAGES FAILED"));
  process.exit(allPassed ? 0 : 1);
}

main().catch(console.error);
