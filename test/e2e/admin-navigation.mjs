import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { chromium } from "playwright";

const baseURL = process.env.E2E_BASE_URL || "http://127.0.0.1:4173";
const server = spawn(
  process.execPath,
  ["node_modules/vite/bin/vite.js", "--host", "127.0.0.1", "--port", "4173", "--strictPort"],
  {
    cwd: `${process.cwd()}/frontend`,
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

/**
 * Expand a sidebar nav group. The 「设置」 group is defaultCollapsed, so ops links
 * (数据迁移 / 系统状态) are unmounted until the header is toggled open.
 * Idempotent when the group is already expanded.
 */
async function expandAdminNavGroup(page, groupLabel) {
  const header = page.getByRole("button", { name: groupLabel, exact: true });
  await header.waitFor({ state: "visible", timeout: 10_000 });

  if (groupLabel === "设置") {
    const hub = page.getByRole("link", { name: "设置中心" });
    if (await hub.isVisible().catch(() => false)) {
      return;
    }
    await header.click();
    await hub.waitFor({ state: "visible", timeout: 5_000 });
    return;
  }

  await header.click();
}

function createMockState() {
  return {
    nextPageId: 101,
    nextScheduledPublicationId: 1,
    nextMigrationJobId: 1,
    pages: [],
    migrationJobs: [],
    migrationStreamAttempts: {},
    migrationStreamAuthFailures: 0,
    authRefreshes: 0,
    validAccessToken: "e2e-access",
    scheduledPublications: [],
    systemStatus: {
      application: { version: "e2e" },
      runtime: {
        goVersion: "go1.24.0",
        os: "linux",
        arch: "amd64",
        cpuCount: 4,
        goroutines: 12,
        uptime: 3600,
      },
      memory: {
        allocMB: 32,
        totalAllocMB: 128,
        sysMB: 64,
        gcPauseMs: 0.2,
      },
      database: {
        type: "sqlite",
        healthy: true,
        status: "healthy",
        openConnections: 1,
        maxOpenConnections: 1,
        inUse: 0,
        idle: 1,
      },
      storage: {
        type: "local",
        healthy: true,
        status: "healthy",
        uploadDirSizeMB: 12.5,
        uploadDirBytes: 13107200,
        mediaCount: 6,
      },
      content: {
        articles: 3,
        pages: 2,
        media: 6,
        users: 1,
      },
    },
  };
}

function createMigrationJob(state, overrides = {}) {
  const now = new Date().toISOString();
  const job = {
    jobId: `mig-${state.nextMigrationJobId++}`,
    source: "markdown",
    phase: "importing",
    total: 3,
    processed: 0,
    succeeded: 0,
    failed: 0,
    errors: [],
    attempt: 1,
    retryable: false,
    startedAt: now,
    finishedAt: null,
    ...overrides,
  };
  state.migrationJobs.unshift(job);
  return job;
}

function completeMigrationJob(job, overrides = {}) {
  Object.assign(job, {
    phase: "done",
    processed: job.total,
    succeeded: Math.max(job.total - job.failed, 0),
    finishedAt: new Date().toISOString(),
    ...overrides,
  });
  return job;
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
        globalConfig: { config: { site: { name: { zh: "Customer Site" } } } },
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
      await json(route, { accessToken: state.validAccessToken, refreshToken: "e2e-refresh" });
      return;
    }
    if (path === "/auth/refresh") {
      const input = request.postDataJSON();
      assert.equal(input.refreshToken, "e2e-refresh");
      state.authRefreshes += 1;
      state.validAccessToken = `e2e-access-refreshed-${state.authRefreshes}`;
      await json(route, {
        accessToken: state.validAccessToken,
        refreshToken: "e2e-refresh",
      });
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

    if (path === "/admin/scheduled-publications" && method === "GET") {
      const resourceType = url.searchParams.get("resourceType");
      const resourceId = Number(url.searchParams.get("resourceId") || 0);
      const status = url.searchParams.get("status");
      const items = state.scheduledPublications.filter((item) =>
        (!resourceType || item.resourceType === resourceType) &&
        (!resourceId || item.resourceId === resourceId) &&
        (!status || item.status === status)
      );
      await json(route, {
        items,
        total: items.length,
        page: Number(url.searchParams.get("page") || 1),
        pageSize: Number(url.searchParams.get("pageSize") || 20),
      });
      return;
    }
    if (path === "/admin/scheduled-publications" && method === "POST") {
      const input = request.postDataJSON();
      const pageItem = state.pages.find((item) => item.id === input.resourceId);
      const now = new Date().toISOString();
      const item = {
        id: state.nextScheduledPublicationId++,
        resourceType: input.resourceType,
        resourceId: input.resourceId,
        title: pageItem?.zhTitle || `#${input.resourceId}`,
        slug: pageItem?.slug || "",
        status: "pending",
        scheduledAt: input.scheduledAt,
        expectedVersion: input.expectedVersion,
        attempts: 0,
        lastError: null,
        createdAt: now,
        updatedAt: now,
        completedAt: null,
      };
      state.scheduledPublications.push(item);
      await json(route, item, 201);
      return;
    }

    const scheduledPublicationMatch = path.match(/^\/admin\/scheduled-publications\/(\d+)$/);
    if (scheduledPublicationMatch && method === "DELETE") {
      const item = state.scheduledPublications.find(
        (candidate) => candidate.id === Number(scheduledPublicationMatch[1]),
      );
      if (!item) {
        await json(route, { error: "scheduled publication not found" }, 404);
        return;
      }
      item.status = "cancelled";
      item.updatedAt = new Date().toISOString();
      item.completedAt = item.updatedAt;
      await json(route, item);
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
    if (path === "/admin/system/status") {
      await json(route, state.systemStatus);
      return;
    }
    if (path === "/admin/migration/import" && method === "POST") {
      const job = createMigrationJob(state, {
        source: "markdown",
        total: 3,
      });
      await json(route, {
        jobId: job.jobId,
        message: "Markdown import queued",
        totalArticles: job.total,
        parseErrors: [],
      }, 202);
      return;
    }
    if (path === "/admin/migration/jobs") {
      await json(route, { jobs: state.migrationJobs });
      return;
    }

    const migrationJobMatch = path.match(/^\/admin\/migration\/jobs\/([^/]+)$/);
    if (migrationJobMatch && method === "GET") {
      if (request.headers().authorization !== `Bearer ${state.validAccessToken}`) {
        await json(route, { error: "unauthorized" }, 401);
        return;
      }
      const job = state.migrationJobs.find((item) => item.jobId === migrationJobMatch[1]);
      if (!job) {
        await json(route, { error: "migration job not found" }, 404);
        return;
      }
      await json(route, job);
      return;
    }

    const migrationRetryMatch = path.match(/^\/admin\/migration\/jobs\/([^/]+)\/retry$/);
    if (migrationRetryMatch && method === "POST") {
      const job = state.migrationJobs.find((item) => item.jobId === migrationRetryMatch[1]);
      if (!job) {
        await json(route, { error: "migration job not found" }, 404);
        return;
      }
      Object.assign(job, {
        phase: "importing",
        processed: job.succeeded,
        failed: 0,
        errors: [],
        attempt: (job.attempt || 1) + 1,
        retryable: false,
        finishedAt: null,
      });
      await json(route, job);
      return;
    }

    const migrationStreamMatch = path.match(/^\/admin\/migration\/jobs\/([^/]+)\/stream$/);
    if (migrationStreamMatch && method === "GET") {
      if (request.headers().authorization !== `Bearer ${state.validAccessToken}`) {
        state.migrationStreamAuthFailures += 1;
        await json(route, { error: "unauthorized" }, 401);
        return;
      }
      const jobId = migrationStreamMatch[1];
      const job = state.migrationJobs.find((item) => item.jobId === jobId);
      if (!job) {
        await route.fulfill({
          status: 404,
          contentType: "text/event-stream",
          body: "event: error\ndata: {\"error\":\"migration job not found\"}\n\n",
        });
        return;
      }

      assert.equal(request.headers().authorization, `Bearer ${state.validAccessToken}`);
      state.migrationStreamAttempts[jobId] = (state.migrationStreamAttempts[jobId] || 0) + 1;
      const progressJob = { ...job, processed: Math.max(1, job.total - 1), succeeded: Math.max(1, job.total - 1) };
      if (state.migrationStreamAttempts[jobId] === 1) {
        Object.assign(job, progressJob);
        await route.fulfill({
          status: 200,
          contentType: "text/event-stream",
          body: `event: progress\ndata: ${JSON.stringify(progressJob)}\n\n`,
        });
        return;
      }

      completeMigrationJob(job);
      await route.fulfill({
        status: 200,
        contentType: "text/event-stream",
        body: `event: progress\ndata: ${JSON.stringify(job)}\n\n`,
      });
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

    // 「设置」组 defaultCollapsed: true — expand before clicking ops items.
    await expandAdminNavGroup(page, "设置");
    await page.getByRole("link", { name: "数据迁移" }).click();
    await page.waitForURL(`${baseURL}/admin/migration`);
    await page.getByRole("heading", { name: "数据迁移" }).waitFor();
    await page.getByText("暂无导入任务", { exact: true }).waitFor();

    // On a settings path the group auto-expands; still assert ops links are available.
    await page.getByRole("link", { name: "系统状态" }).click();
    await page.waitForURL(`${baseURL}/admin/system-status`);
    await page.getByRole("heading", { name: "系统状态" }).waitFor();
    await page.getByText("e2e", { exact: true }).waitFor();
    await page.getByText("sqlite", { exact: true }).waitFor();
    await page.getByRole("link", { name: "数据迁移" }).click();
    await page.waitForURL(`${baseURL}/admin/migration`);
    await page.getByRole("heading", { name: "数据迁移" }).waitFor();
    // Scope to main: system-status also has an AdminButton labeled「刷新」.
    const migrationMain = page.locator("main");
    await migrationMain.getByRole("heading", { name: "导入任务" }).waitFor();
    await migrationMain.getByRole("button", { name: "刷新", exact: true }).waitFor({
      state: "visible",
    });

    createMigrationJob(state, {
      jobId: "mig-failed",
      phase: "failed",
      total: 2,
      processed: 1,
      succeeded: 1,
      failed: 1,
      errors: ["broken front matter"],
      retryable: true,
      finishedAt: new Date().toISOString(),
    });
    await migrationMain.getByRole("button", { name: "刷新", exact: true }).click();
    await migrationMain.getByText("失败", { exact: true }).waitFor();
    await migrationMain.getByRole("button", { name: "重试", exact: true }).click();
    await migrationMain.getByText("成功 2 条，失败 0 条", { exact: true }).waitFor();
    assert.equal(state.migrationStreamAttempts["mig-failed"], 2);

    await page.evaluate(() => {
      localStorage.setItem("inkless.auth.accessToken", "expired-e2e-access");
    });
    await page.getByRole("button", { name: "Markdown ZIP" }).click();
    await page.locator('input[type="file"]').setInputFiles({
      name: "markdown-export.zip",
      mimeType: "application/zip",
      buffer: Buffer.from("e2e markdown zip"),
    });
    await page.getByRole("button", { name: "开始导入", exact: true }).click();
    await page.getByText(/导入任务已创建，任务 ID: mig-\d+，待导入 3 条/).waitFor();
    await page.getByText("成功 3 条，失败 0 条", { exact: true }).waitFor();
    assert.equal(state.migrationStreamAttempts[state.migrationJobs[0].jobId], 2);
    assert.equal(state.migrationStreamAuthFailures, 1);
    assert.equal(state.authRefreshes, 1);

    await page.goto(`${baseURL}/admin/pages`);
    await page.getByRole("heading", { name: "页面管理" }).waitFor();
    // Header + empty-state both render「新建页面」— click the first visible action.
    await page.locator("main").getByRole("button", { name: "新建页面" }).first().click();
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

    await page.goto(`${baseURL}/admin/pages/edit/101`);
    await page.getByRole("button", { name: "安排发布", exact: true }).click();
    const scheduledAt = new Date(Date.now() + 60 * 60 * 1000);
    const localScheduledAt = new Date(
      scheduledAt.getTime() - scheduledAt.getTimezoneOffset() * 60_000,
    ).toISOString().slice(0, 16);
    await page.locator('input[type="datetime-local"]').fill(localScheduledAt);
    await Promise.all([
      page.waitForResponse((response) =>
        response.url().endsWith("/admin/scheduled-publications") &&
        response.request().method() === "POST" &&
        response.ok()
      ),
      page.locator('button[type="submit"]').filter({ hasText: "Schedule" }).click(),
    ]);
    await page.getByText("等待发布", { exact: true }).waitFor();
    assert.equal(state.scheduledPublications[0].expectedVersion, state.pages[0].draftVersion);

    await page.getByRole("link", { name: "定时发布" }).click();
    await page.waitForURL(`${baseURL}/admin/scheduled-publications`);
    await page.getByRole("heading", { name: "定时发布", exact: true }).waitFor();
    await page.getByText("发布页", { exact: true }).waitFor();
    await Promise.all([
      page.waitForResponse((response) =>
        response.url().endsWith("/admin/scheduled-publications/1") &&
        response.request().method() === "DELETE" &&
        response.ok()
      ),
      page.getByRole("button", { name: "取消", exact: true }).click(),
    ]);
    await page.getByText("暂无定时发布任务", { exact: true }).waitFor();
    assert.equal(state.scheduledPublications[0].status, "cancelled");

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
    assert.equal(await editorPage.locator("main").getByRole("button", { name: "新建页面" }).count(), 0);
    assert.equal(await editorPage.locator("main").getByRole("button", { name: "删除", exact: true }).count(), 0);
    // exact: avoid matching the account menu ("editor 编辑")
    await editorPage.locator("main").getByRole("button", { name: "编辑", exact: true }).click();
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
