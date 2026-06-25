import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import { fileURLToPath } from "node:url";
import type { Plugin } from "vite";

function e2eWailsRuntime(runtimePath: string): Plugin {
  return {
    name: "whisk-e2e-wails-runtime",
    enforce: "pre",
    resolveId(id, importer) {
      if (id !== "@wailsio/runtime" || !importer) return undefined;
      if (!importer.includes("/frontend/src/") && !importer.includes("/frontend/bindings/")) {
        return undefined;
      }
      return runtimePath;
    },
  };
}

export default defineConfig(({ mode }) => {
  const e2eRuntime = fileURLToPath(new URL("./e2e/wailsRuntimeFake.ts", import.meta.url));

  return {
    server: {
      host: "127.0.0.1",
      port: Number(process.env.WAILS_VITE_PORT) || 9245,
      strictPort: true,
    },
    plugins: [mode === "e2e" ? e2eWailsRuntime(e2eRuntime) : null, svelte(), tailwindcss(), wails("./bindings")],
  };
});
