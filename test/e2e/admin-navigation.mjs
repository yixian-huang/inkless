import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { chromium } from "playwright";

const baseURL = process.env.E2E_BASE_URL || "http://127.0.0.1:4173";
const server = spawn(
  "pnpm",
  ["-C", "frontend", "dev", "--host", "127.0.0.1", "--port", "4173", "--strictPort"],
  {
    cwd: process.cwd(),
    env: { ...process.env, NODE_ENV: "test" },
    stdio: ["ignore", "pipe", "pipe"],
  },
);

let serverOutput = "";
server.stdout.on("data", (chunk) => {
  serverOutput += chunk.toString();
});
server.stderr.on("data", (chunk) => {
  serverOutput += chunk.toString();
});

async function waitForServer() {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(baseURL);
      if (response.ok) return;
    } catch {
      // Vite is still starting.
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`Vite did not start in time.\n${serverOutput}`);
}

function json(route, body, status = 200) {
  return route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });
}

async function mockAdminAPI(page) {
  await page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;

    if (request.resourceType() === "document" || url.origin !== baseURL) {
      await route.continue();
      return;
    }

    if (path === "/setup/status") {
      await json(route, { installed: true, databaseType: "sqlite" });
      return;
    }
    if (path === "/public/bootstrap") {
      await json(route, {
        activeTheme: {},
        themeTokens: {},
        themePages: [],
        globalConfig: { config: { site: { name: { zh: "Impress" } } } },
        features: {},
      });
      return;
    }
    if (path === "/auth/login") {
      await json(route, { accessToken: "e2e-access", refreshToken: "e2e-refresh" });
      return;
    }
    if (path === "/auth/me") {
      await json(route, {
        id: "1",
        username: "admin",
        role: "admin",
        isSuperAdmin: true,
        permissions: ["*:*"],
      });
      return;
    }
    if (path === "/admin/analytics/summary") {
      await json(route, {
        pages: [],
        totals: { today: 0, last7d: 0, last30d: 0 },
      });
      return;
    }
    if (path === "/admin/pages") {
      await json(route, { items: [] });
      return;
    }
    if (path === "/admin/articles") {
      await json(route, { items: [], total: 0, page: 1, pageSize: 10 });
      return;
    }
    if (path === "/admin/media") {
      await json(route, { items: [], total: 0, page: 1, pageSize: 20 });
      return;
    }
    if (path === "/admin/migration/jobs") {
      await json(route, { jobs: [] });
      return;
    }
    if (path.startsWith("/admin/") || path.startsWith("/public/")) {
      await json(route, { items: [], total: 0 });
      return;
    }

    await route.continue();
  });
}

async function run() {
  await waitForServer();
  const browser = await chromium.launch({ headless: true });

  try {
    const page = await browser.newPage();
    const pageErrors = [];
    page.on("pageerror", (error) => pageErrors.push(error));
    await mockAdminAPI(page);

    await page.goto(`${baseURL}/admin`);
    await page.waitForURL(`${baseURL}/admin/login`);
    assert.equal(page.url(), `${baseURL}/admin/login`);

    await page.getByLabel("用户名").fill("admin");
    await page.getByLabel("密码").fill("admin123");
    await page.getByRole("button", { name: "登录", exact: true }).click();

    await page.waitForURL(`${baseURL}/admin`);
    assert.equal(page.url(), `${baseURL}/admin`);
    await page.getByRole("heading", { name: "仪表盘" }).waitFor();

    await page.getByRole("link", { name: "页面管理" }).click();
    await page.waitForURL(`${baseURL}/admin/pages`);
    await page.getByRole("heading", { name: "页面管理" }).waitFor();

    await page.getByRole("link", { name: "文章管理" }).click();
    await page.waitForURL(`${baseURL}/admin/articles`);
    await page.getByRole("heading", { name: "文章管理" }).waitFor();

    await page.getByRole("link", { name: "数据迁移" }).click();
    await page.waitForURL(`${baseURL}/admin/migration`);
    await page.getByRole("heading", { name: "数据迁移" }).waitFor();

    assert.equal(pageErrors.length, 0, `Unexpected page errors: ${pageErrors.join("\n")}`);
    console.log("Admin release-chain E2E passed");
  } finally {
    await browser.close();
  }
}

try {
  await run();
} finally {
  server.kill("SIGTERM");
}
