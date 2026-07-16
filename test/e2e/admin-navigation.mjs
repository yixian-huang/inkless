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

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function createMockState() {
  return {
    nextPageId: 101,
    pages: [],
  };
}

function publicPageFacts(page) {
  return {
    id: page.id,
    slug: page.slug,
    title: { zh: page.zhTitle, en: page.enTitle },
    description: { zh: page.zhDescription || "", en: page.enDescription || "" },
    mode: page.mode,
    sortOrder: page.sortOrder,
    showInNav: page.showInNav,
    parentId: page.parentId,
    status: page.status,
    publishedVersion: page.publishedVersion,
  };
}

async function mockAdminAPI(page, state, currentUser = {
  id: "1",
  username: "admin",
  role: "admin",
  isSuperAdmin: true,
  permissions: ["*:*"],
}) {
  await page.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

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
        unifiedPages: state.pages
          .filter((item) => item.status === "published" && item.publishedConfig)
          .map(publicPageFacts),
        globalConfig: { config: { site: { name: { zh: "Impress" } } } },
        features: {},
      });
      return;
    }
    if (path === "/public/menu") {
      await json(route, {
        id: 1,
        name: "Primary",
        slug: "primary",
        isPrimary: true,
        items: [],
        createdAt: "2026-07-16T00:00:00Z",
        updatedAt: "2026-07-16T00:00:00Z",
      });
      return;
    }
    if (path === "/auth/login") {
      await json(route, { accessToken: "e2e-access", refreshToken: "e2e-refresh" });
      return;
    }
    if (path === "/auth/me") {
      await json(route, currentUser);
      return;
    }
    if (path === "/admin/analytics/summary") {
      await json(route, {
        pages: [],
        totals: { today: 0, last7d: 0, last30d: 0 },
      });
      return;
    }
    if (path === "/admin/pages" && method === "GET") {
      await json(route, { items: state.pages });
      return;
    }
    if (path === "/admin/pages" && method === "POST") {
      const input = request.postDataJSON();
      const now = new Date().toISOString();
      const item = {
        id: state.nextPageId++,
        slug: input.slug,
        zhTitle: input.zhTitle || "",
        enTitle: input.enTitle || "",
        zhDescription: "",
        enDescription: "",
        mode: input.mode || "composable",
        templateId: input.templateId,
        status: "draft",
        sortOrder: input.sortOrder || 0,
        showInNav: Boolean(input.showInNav),
        parentId: input.parentId,
        publishedVersion: 0,
        draftVersion: 1,
        draftConfig: input.draftConfig || { sections: [] },
        publishedConfig: null,
        createdAt: now,
        updatedAt: now,
      };
      state.pages.push(item);
      await json(route, item, 201);
      return;
    }

    const draftMatch = path.match(/^\/admin\/pages\/(\d+)\/draft$/);
    if (draftMatch) {
      const item = state.pages.find((candidate) => candidate.id === Number(draftMatch[1]));
      if (!item) {
        await json(route, { error: "page not found" }, 404);
        return;
      }
      if (method === "GET") {
        await json(route, {
          id: item.id,
          slug: item.slug,
          draftConfig: item.draftConfig,
          draftVersion: item.draftVersion,
          publishedVersion: item.publishedVersion,
          updatedAt: item.updatedAt,
        });
        return;
      }
      if (method === "PUT") {
        const expectedVersion = Number(request.headers()["if-match"]);
        if (expectedVersion !== item.draftVersion) {
          await json(route, {
            error: "draft version conflict",
            currentVersion: item.draftVersion,
          }, 409);
          return;
        }
        const input = request.postDataJSON();
        item.draftConfig = input.draftConfig;
        item.draftVersion += 1;
        item.updatedAt = new Date().toISOString();
        await json(route, { id: item.id, draftVersion: item.draftVersion });
        return;
      }
    }

    const publishMatch = path.match(/^\/admin\/pages\/(\d+)\/publish$/);
    if (publishMatch && method === "POST") {
      const item = state.pages.find((candidate) => candidate.id === Number(publishMatch[1]));
      const input = request.postDataJSON();
      if (!item) {
        await json(route, { error: "page not found" }, 404);
        return;
      }
      if (input.expectedDraftVersion !== item.draftVersion) {
        await json(route, { error: "draft version conflict" }, 409);
        return;
      }
      item.status = "published";
      item.publishedVersion += 1;
      item.publishedConfig = clone(item.draftConfig);
      item.updatedAt = new Date().toISOString();
      await json(route, item);
      return;
    }

    const unpublishMatch = path.match(/^\/admin\/pages\/(\d+)\/unpublish$/);
    if (unpublishMatch && method === "POST") {
      const item = state.pages.find((candidate) => candidate.id === Number(unpublishMatch[1]));
      if (!item) {
        await json(route, { error: "page not found" }, 404);
        return;
      }
      item.status = "draft";
      item.publishedVersion = 0;
      item.publishedConfig = null;
      item.updatedAt = new Date().toISOString();
      await json(route, { message: "unpublished" });
      return;
    }

    const versionsMatch = path.match(/^\/admin\/pages\/(\d+)\/versions$/);
    if (versionsMatch && method === "GET") {
      const item = state.pages.find((candidate) => candidate.id === Number(versionsMatch[1]));
      await json(route, {
        items: item ? [{
          id: 1,
          pageId: item.id,
          version: 1,
          createdAt: item.updatedAt,
        }] : [],
        total: item ? 1 : 0,
        page: 1,
        pageSize: 20,
      });
      return;
    }

    const pageMatch = path.match(/^\/admin\/pages\/(\d+)$/);
    if (pageMatch) {
      const item = state.pages.find((candidate) => candidate.id === Number(pageMatch[1]));
      if (!item) {
        await json(route, { error: "page not found" }, 404);
        return;
      }
      if (method === "GET") {
        await json(route, item);
        return;
      }
      if (method === "PUT") {
        Object.assign(item, request.postDataJSON(), {
          updatedAt: new Date().toISOString(),
        });
        await json(route, item);
        return;
      }
    }

    const publicPageMatch = path.match(/^\/public\/pages\/([^/]+)$/);
    if (publicPageMatch && method === "GET") {
      const item = state.pages.find(
        (candidate) =>
          candidate.slug === publicPageMatch[1] &&
          candidate.status === "published" &&
          candidate.publishedConfig,
      );
      if (!item) {
        await json(route, { error: "page not found" }, 404);
        return;
      }
      await json(route, {
        ...publicPageFacts(item),
        title: item.zhTitle,
        description: item.zhDescription,
        publishedConfig: item.publishedConfig,
      });
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
    const state = createMockState();
    page.on("pageerror", (error) => pageErrors.push(error));
    await mockAdminAPI(page, state);

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

    await page.goto(`${baseURL}/admin/pages`);
    await page.getByRole("button", { name: "新建页面" }).click();
    await page.waitForURL(`${baseURL}/admin/pages/new`);
    await page.getByLabel("URL 路径 (slug)").fill("launch-page");
    await page.getByLabel("标题 (中文)").fill("发布页");
    await page.getByLabel("标题 (English)").fill("Launch Page");
    await page.getByLabel("显示在导航").check();
    await page.getByRole("button", { name: "JSON 模式" }).click();
    await page.getByRole("textbox", { name: "区块配置 (JSON 数组)" }).fill(JSON.stringify([
      {
        id: "launch-copy",
        type: "rich-text",
        variant: "default",
        locked: false,
        data: { content: "Wave 1 published page" },
        settings: {},
      },
    ]));
    await page.getByRole("button", { name: "创建", exact: true }).click();
    await page.waitForURL(`${baseURL}/admin/pages/edit/101`);

    await page.getByRole("button", { name: "发布", exact: true }).click();
    await page.getByText("已发布", { exact: true }).waitFor();

    await page.goto(`${baseURL}/launch-page`);
    await page.getByText("Wave 1 published page", { exact: true }).waitFor();
    await page.getByRole("navigation").getByRole("link", { name: "发布页" }).waitFor();

    await page.goto(`${baseURL}/admin/pages/edit/101`);
    await page.getByLabel("显示在导航").uncheck();
    await Promise.all([
      page.waitForResponse((response) =>
        response.url().endsWith("/admin/pages/101") &&
        response.request().method() === "PUT" &&
        response.ok()
      ),
      page.getByRole("button", { name: "保存页面信息", exact: true }).click(),
    ]);
    assert.equal(state.pages[0].showInNav, false);

    await page.goto(`${baseURL}/launch-page`);
    await page.getByText("Wave 1 published page", { exact: true }).waitFor();
    assert.equal(await page.getByRole("link", { name: "发布页" }).count(), 0);

    await page.goto(`${baseURL}/admin/pages/edit/101`);
    await Promise.all([
      page.waitForResponse((response) =>
        response.url().endsWith("/admin/pages/101/unpublish") &&
        response.request().method() === "POST" &&
        response.ok()
      ),
      page.getByRole("button", { name: "下线", exact: true }).click(),
    ]);
    assert.equal(state.pages[0].status, "draft");
    await page.getByRole("button", { name: "发布", exact: true }).waitFor();

    await page.goto(`${baseURL}/launch-page`);
    await page.getByRole("heading", { name: "404", exact: true }).waitFor();

    assert.equal(pageErrors.length, 0, `Unexpected page errors: ${pageErrors.join("\n")}`);

    const editorPage = await browser.newPage();
    await mockAdminAPI(editorPage, state, {
      id: "2",
      username: "editor",
      role: "editor",
      isSuperAdmin: false,
      permissions: ["pages:read", "pages:update"],
    });
    await editorPage.goto(`${baseURL}/admin`);
    await editorPage.getByLabel("用户名").fill("editor");
    await editorPage.getByLabel("密码").fill("editor123");
    await editorPage.getByRole("button", { name: "登录", exact: true }).click();
    await editorPage.waitForURL(`${baseURL}/admin`);

    await editorPage.goto(`${baseURL}/admin/pages`);
    await editorPage.getByRole("heading", { name: "页面管理" }).waitFor();
    assert.equal(await editorPage.getByRole("button", { name: "新建页面" }).count(), 0);
    assert.equal(await editorPage.getByRole("button", { name: "删除" }).count(), 0);
    await editorPage.getByRole("button", { name: "编辑" }).click();
    await editorPage.waitForURL(`${baseURL}/admin/pages/edit/101`);
    await editorPage.getByRole("button", { name: "保存草稿" }).waitFor();
    assert.equal(await editorPage.getByRole("button", { name: "发布", exact: true }).count(), 0);
    assert.equal(await editorPage.getByRole("button", { name: "下线", exact: true }).count(), 0);
    await editorPage.getByRole("button", { name: "版本历史" }).click();
    await editorPage.getByRole("heading", { name: "版本历史" }).waitFor();
    assert.equal(await editorPage.getByRole("button", { name: "回滚", exact: true }).count(), 0);
    await editorPage.close();

    console.log("Admin navigation and unified-page release-chain E2E passed");
  } finally {
    await browser.close();
  }
}

try {
  await run();
} finally {
  server.kill("SIGTERM");
}
