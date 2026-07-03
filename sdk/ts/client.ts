import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

const tokenFileName = "control-token";

export function stateDir() {
  const xdgStateHome = process.env.XDG_STATE_HOME;
  if (xdgStateHome) return join(xdgStateHome, "whisk");
  if (process.platform === "win32" && process.env.LOCALAPPDATA) return join(process.env.LOCALAPPDATA, "whisk", "state");
  return join(homedir(), ".local", "state", "whisk");
}

export function controlTokenPath() {
  return join(stateDir(), tokenFileName);
}

export function readControlToken() {
  return readFileSync(controlTokenPath(), "utf8").trim();
}

export function controlAuthHeaders() {
  return { Authorization: `Bearer ${readControlToken()}` };
}

export function whiskdClientOptions(options: { baseUrl?: string; headers?: Record<string, string> } = {}) {
  return {
    ...options,
    baseUrl: options.baseUrl ?? process.env.WHISKD_URL ?? "http://127.0.0.1:8787",
    headers: {
      ...(options.headers ?? {}),
      ...controlAuthHeaders(),
    },
  };
}
