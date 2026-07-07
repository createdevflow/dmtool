import type { NextConfig } from "next";

// Pin Turbopack's workspace root to the frontend/ folder. Without
// this, Next 16 walks up to the repo root looking for tailwindcss
// and similar module roots; when it finds neither node_modules nor
// a lockfile there, it fails to resolve "@import 'tailwindcss'".
// We hit this in dev mode (the production build path was less
// affected because Next looked inward from the build cwd).
//
// process.cwd() is the source of truth at config-load time: Next
// runs `next dev` from the project root, which is frontend/. The
// earlier `import.meta.url` variant was ESM-only and broke when
// Next compiled next.config.ts to CJS (ReferenceError: exports).
const turbopackRoot = process.cwd();

const nextConfig: NextConfig = {
  turbopack: {
    root: turbopackRoot,
  },
};

export default nextConfig;
