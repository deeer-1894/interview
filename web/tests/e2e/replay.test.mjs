import test from "node:test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { setTimeout as delay } from "node:timers/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(__dirname, "../..");
const backendUrl = process.env.API_BASE_URL || "http://127.0.0.1:18081";
const frontendUrl = process.env.FRONTEND_URL || "http://127.0.0.1:4173";

function spawnManaged(command, args, options = {}) {
  const child = spawn(command, args, {
    cwd: options.cwd,
    env: options.env,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "";
  const append = (chunk) => {
    output += chunk.toString();
    if (output.length > 12000) {
      output = output.slice(-12000);
    }
  };
  child.stdout?.on("data", append);
  child.stderr?.on("data", append);
  return {
    child,
    getOutput: () => output.trim(),
  };
}

async function stopManaged(proc) {
  if (!proc) {
    return;
  }
  if (proc.exitCode === null) {
    proc.kill("SIGINT");
  }
  await Promise.race([
    new Promise((resolve) => proc.once("exit", resolve)),
    delay(5000).then(() => {
      if (proc.exitCode === null) {
        proc.kill("SIGKILL");
      }
    }),
  ]);
}

async function waitForHttpOk(url, label, timeoutMs = 30000) {
  const started = Date.now();
  let lastError = "";
  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
      lastError = `${response.status} ${response.statusText}`;
    } catch (error) {
      lastError = error instanceof Error ? error.message : String(error);
    }
    await delay(1000);
  }
  throw new Error(`${label} not ready: ${lastError}`);
}

async function requestJSON(method, url, body) {
  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await response.text();
  let payload = null;
  try {
    payload = text ? JSON.parse(text) : null;
  } catch {
    payload = text;
  }
  if (!response.ok) {
    throw new Error(`${method} ${url} failed: ${response.status} ${JSON.stringify(payload)}`);
  }
  return payload;
}

test("review workspace replay smoke", { timeout: 180000 }, async () => {
  let frontend;
  let browser;
  let page;
  let conversationId = "";
  const workspaceTitle = `UI Smoke PW ${Date.now()}`;

  try {
    await waitForHttpOk(`${backendUrl}/api/health`, "backend", 30000);

    const interview = await requestJSON("POST", `${backendUrl}/api/interview`, {
      prompt: "请模拟一场很短的 Go 面试，只问一个问题并结束。",
      config: {
        persona: "supportive",
        level: "初级",
        focus: "generalist",
        mode: "standard",
        timeBudget: "15 分钟",
        outputStyle: "interview_plus_score",
      },
    });
    conversationId = interview.conversationId;
    assert.ok(conversationId, "conversationId should be returned");

    await requestJSON("PATCH", `${backendUrl}/api/conversations/${conversationId}`, {
      title: workspaceTitle,
      pinned: true,
    });

    frontend = spawnManaged("npm", ["run", "dev", "--", "--host", "127.0.0.1", "--port", "4173"], {
      cwd: webDir,
      env: {
        ...process.env,
        VITE_API_PROXY_TARGET: backendUrl,
      },
    });
    await waitForHttpOk(frontendUrl, "frontend");

    browser = await chromium.launch({ headless: true });
    page = await browser.newPage();
    await page.goto(frontendUrl, { waitUntil: "domcontentloaded" });

    await page.getByText(workspaceTitle).first().waitFor({ timeout: 30000 });
    await page.getByText(workspaceTitle).first().click();
    await page.getByText("本场结果").first().waitFor({ timeout: 30000 });
    await page.getByText("总分").first().waitFor({ timeout: 30000 });

    const traceToggle = page.getByText("展开追问树").first();
    if (await traceToggle.count()) {
      await traceToggle.click();
      await page.getByText("小型追问树视图").first().waitFor({ timeout: 30000 });
    }

    await page.getByText("展开画像雷达").first().click();
    await page.getByText("画像雷达图").first().waitFor({ timeout: 30000 });
  } finally {
    if (browser) {
      await browser.close();
    }
    if (conversationId) {
      try {
        await fetch(`${backendUrl}/api/conversations/${conversationId}`, { method: "DELETE" });
      } catch {
        // Cleanup should not mask test failures.
      }
    }
    await stopManaged(frontend?.child);
  }
});
