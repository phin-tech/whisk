import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync } from "node:fs";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { TestProject } from "vitest/node";

// Provide the live daemon base URL to the suite. Empty string => no binary, the
// tests skip themselves. Boots whisk on an ephemeral port with isolated XDG
// state so it never touches real developer session state.
declare module "vitest" {
  interface ProvidedContext {
    baseUrl: string;
    stateHome: string;
  }
}

function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = createServer();
    srv.on("error", reject);
    srv.listen(0, "127.0.0.1", () => {
      const addr = srv.address();
      const port = typeof addr === "object" && addr ? addr.port : 0;
      srv.close(() => resolve(port));
    });
  });
}

export default async function setup(project: TestProject) {
  const binary = process.env.WHISKD_BIN;
  if (!binary) {
    project.provide("baseUrl", "");
    project.provide("stateHome", "");
    return;
  }

  const port = await freePort();
  const addr = `127.0.0.1:${port}`;
  const state = mkdtempSync(join(tmpdir(), "whiskd-ts-"));
  const stateHome = join(state, "state");
  const proc = spawn(binary, ["daemon", "run", "-addr", addr], {
    stdio: "pipe",
    env: {
      ...process.env,
      WHISKD_ADDR: addr,
      XDG_CONFIG_HOME: join(state, "config"),
      XDG_DATA_HOME: join(state, "data"),
      XDG_STATE_HOME: stateHome,
      XDG_CACHE_HOME: join(state, "cache"),
    },
  });

  const url = `http://${addr}`;
  const tokenPath = join(stateHome, "whisk", "control-token");
  const deadline = Date.now() + 15_000;
  while (Date.now() < deadline) {
    if (proc.exitCode !== null) {
      throw new Error(`daemon exited early with code ${proc.exitCode}`);
    }
    try {
      if (existsSync(tokenPath)) {
        const token = readFileSync(tokenPath, "utf8").trim();
        const res = await fetch(`${url}/v1/compat`, { headers: { Authorization: `Bearer ${token}` } });
        if (res.ok) break;
      }
    } catch {
      await new Promise((r) => setTimeout(r, 100));
    }
  }

  project.provide("baseUrl", url);
  project.provide("stateHome", stateHome);

  return () => {
    proc.kill("SIGTERM");
  };
}
