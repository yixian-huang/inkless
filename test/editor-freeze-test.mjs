/**
 * Test: Editor Image Insert/Delete Freeze Detection
 * Tests on production: http://47.93.134.202:18090/admin
 */
import { chromium } from "playwright";

const BASE = "http://47.93.134.202:18090";
const TIMEOUT = 15000;

async function run() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  page.setDefaultTimeout(TIMEOUT);

  // Collect console errors
  const errors = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(msg.text());
  });

  try {
    // 1. Login
    console.log("1. Logging in...");
    await page.goto(`${BASE}/admin/login`);
    await page.fill('input[type="text"], input[name="username"]', "admin");
    await page.fill('input[type="password"]', "admin123");
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/admin/, { timeout: 10000 });
    console.log("   Login OK");

    // 2. Navigate to article editor via admin UI (nginx proxies /admin/articles/* to backend)
    console.log("2. Navigating to article list...");
    // Click "文章" in the sidebar
    const articleLink = page.locator('a[href="/admin/articles"]');
    await articleLink.click({ timeout: 5000 }).catch(() => {
      // If no sidebar link, try navigating directly
      console.log("   No sidebar article link, trying nav");
    });
    await page.waitForTimeout(2000);
    await page.screenshot({ path: "/tmp/editor-step2.png" });

    // Find and click edit button for the article
    console.log("   Looking for edit link...");
    const editLink = page.locator("text=Edit").first();
    await editLink.click({ timeout: 5000 });
    await page.waitForTimeout(3000);
    await page.screenshot({ path: "/tmp/editor-step3.png" });

    // Wait for editor to load
    const editorSel = ".tiptap, .ProseMirror";
    try {
      await page.waitForSelector(editorSel, { timeout: 10000 });
      console.log("   Editor loaded");
    } catch {
      console.log("   Editor not found, checking page state...");
      console.log("   Current URL:", page.url());
      const bodyText = await page.locator("body").innerText();
      console.log("   Body text (first 200):", bodyText.substring(0, 200));
      throw new Error("Could not load editor");
    }

    // 3. Verify existing image
    const existingImages = await page.locator(".tiptap img").count();
    console.log(`   Existing images in editor: ${existingImages}`);

    // 4. Click the "图片" toolbar button to open image picker
    console.log("3. Opening image picker...");
    const t0 = Date.now();
    const imgBtn = page.locator('button[title="插入图片"]');
    await imgBtn.click();
    const pickerOpenTime = Date.now() - t0;
    console.log(`   Image picker button clicked in ${pickerOpenTime}ms`);

    // Wait for the image picker modal to appear
    await page.waitForSelector('[class*="fixed"]', { timeout: 5000 });
    console.log("   Image picker modal visible");

    // 5. Wait for images to load in the picker
    await page.waitForTimeout(2000);
    const pickerImages = await page.locator('[class*="fixed"] img').count();
    console.log(`   Images in picker: ${pickerImages}`);

    if (pickerImages === 0) {
      console.log("   No images in picker, skipping insert test");
    } else {
      // 6. Select the first image
      console.log("4. Selecting image...");
      const t1 = Date.now();

      const firstImg = page.locator('[class*="fixed"] img').first();
      await firstImg.click({ timeout: 5000 });
      const insertTime = Date.now() - t1;
      console.log(`   Image click completed in ${insertTime}ms`);

      // 7. Check if page is still responsive
      const t2 = Date.now();
      try {
        await page.waitForSelector(".tiptap", { timeout: 5000 });
        const responseTime = Date.now() - t2;
        console.log(`   Page responsive after insert: ${responseTime}ms`);

        const newImageCount = await page.locator(".tiptap img").count();
        console.log(`   Images after insert: ${newImageCount}`);

        if (newImageCount > existingImages) {
          console.log("   SUCCESS: Image inserted without freeze!");
        } else {
          console.log("   Image count unchanged - picker may need different interaction");
        }
      } catch (e) {
        console.log(`   FREEZE DETECTED: Page unresponsive after ${Date.now() - t2}ms`);
        console.log(`   Error: ${e.message}`);
      }
    }

    // 8. Test deleting an image
    console.log("5. Testing image deletion...");
    const imagesBeforeDelete = await page.locator(".tiptap img").count();
    if (imagesBeforeDelete > 0) {
      const img = page.locator(".tiptap img").first();
      await img.click();
      await page.waitForTimeout(300);

      const t3 = Date.now();
      await page.keyboard.press("Backspace");

      try {
        await page.waitForTimeout(500);
        const responseAfterDelete = Date.now() - t3;
        const imagesAfterDelete = await page.locator(".tiptap img").count();
        console.log(`   Delete completed in ${responseAfterDelete}ms`);
        console.log(`   Images after delete: ${imagesAfterDelete}`);
        if (imagesAfterDelete < imagesBeforeDelete) {
          console.log("   SUCCESS: Image deleted without freeze!");
        }
      } catch (e) {
        console.log(`   FREEZE DETECTED on delete: ${e.message}`);
      }
    }

    // 9. Report console errors
    if (errors.length > 0) {
      console.log("\nConsole errors:");
      errors.forEach((e) => console.log(`  - ${e}`));
    }

    console.log("\n=== Test Complete ===");
  } catch (e) {
    console.error("Test failed:", e.message);
  } finally {
    await browser.close();
  }
}

run();
