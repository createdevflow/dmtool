import { defineConfig } from "vitest/config";
import { resolve } from "node:path";

export default defineConfig({
  resolve: {
    alias: {
      // Mirror tsconfig paths so tests can use @/ imports the same
      // way the rest of the codebase does.
      "@": resolve(__dirname),
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    include: [
      "**/*.test.ts",
      "**/*.test.tsx",
      // The phase8 nav audit script lives under backend/ and is
      // not part of this config — node scripts aren't vitest tests.
    ],
    exclude: ["node_modules", ".next", "dist"],
  },
});
