#!/usr/bin/env node
// nav_audit.js — phase8 nav completeness check.
//
// 1. Parse navGroups out of frontend/components/dashboard/sidebar.tsx
//    and verify every href resolves to an existing page.tsx under
//    app/(dashboard)/... (dynamic segments like [mode]/[id] accepted).
// 2. Stale-route grep: no code reference to /social-insights,
//    /content-ai, or bare /settings (only /projects/settings is fine).
//    Body-text occurrences of "project settings" are NOT flagged.
// 3. Informational: every dashboard page on disk is listed under
//    either a navGroups href or as a drill-down under one.
//
// Run from frontend/ via `node ../backend/_tools/phase8_verify/nav_audit.js`.

const fs = require("fs");
const path = require("path");

const ROOT = path.resolve(__dirname, "..", "..", "..", "frontend");
const SIDEBAR = path.join(ROOT, "components", "dashboard", "sidebar.tsx");
const APP_DIR = path.join(ROOT, "app", "(dashboard)");

function extractNavGroupHrefs(src) {
  const header = src.indexOf("export const navGroups");
  if (header < 0) throw new Error("navGroups header not found");
  const equals = src.indexOf("=", header);
  if (equals < 0) throw new Error("navGroups `=` not found");
  const arrStart = src.indexOf("[", equals);
  if (arrStart < 0) throw new Error("navGroups array open not found");
  let depth = 0;
  let end = -1;
  for (let i = arrStart; i < src.length; i++) {
    const c = src[i];
    if (c === "[") depth++;
    else if (c === "]") {
      depth--;
      if (depth === 0) { end = i; break; }
    }
  }
  if (end < 0) throw new Error("navGroups array close not found");
  const body = src.slice(arrStart, end + 1);
  const re = /href:\s*"([^"]+)"/g;
  const out = [];
  let m;
  while ((m = re.exec(body)) !== null) out.push(m[1]);
  return out;
}

function hrefResolvesToPage(href) {
  const urlPath = href.split("?")[0].split("#")[0];
  if (urlPath === "/") return true;
  const segs = urlPath.split("/").filter(Boolean);
  let cur = APP_DIR;
  for (let i = 0; i < segs.length; i++) {
    const seg = segs[i];
    if (i === segs.length - 1) {
      const leaf = path.join(cur, seg, "page.tsx");
      if (fs.existsSync(leaf)) return true;
      try {
        const entries = fs.readdirSync(path.join(cur, seg), {
          withFileTypes: true,
        });
        for (const e of entries) {
          if (e.isFile() && e.name === "page.tsx") return true;
        }
      } catch {}
      return false;
    }
    const next = path.join(cur, seg);
    if (!fs.existsSync(next)) return false;
    cur = next;
  }
  return false;
}

function allDashboardPages() {
  const out = [];
  function walk(dir, prefix) {
    let entries;
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); }
    catch { return; }
    for (const e of entries) {
      const full = path.join(dir, e.name);
      const url = prefix + "/" + e.name;
      if (e.isDirectory()) walk(full, url);
      else if (e.isFile && e.name === "page.tsx") out.push(url);
    }
  }
  walk(APP_DIR, "");
  return out.sort();
}

function checkStaleRoutes(failures) {
  const ignoreDirs = new Set(["node_modules", ".next", "_tools", ".git"]);
  const codeFiles = [];
  function walk(dir) {
    let entries;
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); }
    catch { return; }
    for (const e of entries) {
      if (ignoreDirs.has(e.name)) continue;
      const full = path.join(dir, e.name);
      if (e.isDirectory()) walk(full);
      else if (
        e.isFile &&
        (e.name.endsWith(".ts") || e.name.endsWith(".tsx"))
      ) {
        codeFiles.push(full);
      }
    }
  }
  walk(ROOT);

  const navBanned = [
    { pat: "/social-insights", note: "old flat command-menu href" },
    { pat: "/content-ai", note: "old flat command-menu href" },
  ];
  // Bare /settings as a nav ref: /projects/settings is allowed.
  const bareSettingsRe = /(['"`])\/settings(?!\/)/;

  for (const f of codeFiles) {
    const src = fs.readFileSync(f, "utf8");
    const lines = src.split("\n");
    lines.forEach((line, idx) => {
      const trimmed = line.trim();
      const isComment =
        trimmed.startsWith("//") ||
        trimmed.startsWith("/*") ||
        trimmed.startsWith("*");
      if (isComment) return;
      for (const { pat, note } of navBanned) {
        if (line.includes(pat)) {
          failures.push(
            `stale route ${pat} (${note}) at ${f}:${idx + 1}: ${line.trim()}`
          );
        }
      }
      if (bareSettingsRe.test(line) && !/\/projects\/settings/.test(line)) {
        failures.push(
          `stale route /settings at ${f}:${idx + 1}: ${line.trim()}`
        );
      }
    });
  }
}

function main() {
  const failures = [];
  let sidebarSrc;
  try { sidebarSrc = fs.readFileSync(SIDEBAR, "utf8"); }
  catch (e) {
    console.error("Cannot read sidebar.tsx:", e.message);
    process.exit(2);
  }
  const hrefs = Array.from(new Set(extractNavGroupHrefs(sidebarSrc)));

  console.log(`Nav hrefs discovered in sidebar.tsx: ${hrefs.length}`);
  for (const href of hrefs) {
    const ok = hrefResolvesToPage(href);
    console.log(`  [${ok ? "OK  " : "FAIL"}] ${href}`);
    if (!ok) {
      failures.push(
        `navGroups contains ${href} but no app/(dashboard)/...${href}/page.tsx resolves`
      );
    }
  }

  checkStaleRoutes(failures);

  // Informational: drill-downs.
  const dashboardPages = allDashboardPages();
  const info = [];
  for (const p of dashboardPages) {
    const route = p.replace(/\/page\.tsx$/, "");
    const matched = hrefs.some(
      (h) =>
        h === route ||
        (h === "/dashboard" && route.startsWith("/dashboard"))
    );
    if (!matched) info.push(p);
  }
  console.log("");
  console.log(
    "Dashboard pages not in main nav (drill-downs, informational):"
  );
  if (info.length === 0) console.log("  (none)");
  else for (const i of info) console.log(`  [INFO] ${i}`);

  if (failures.length > 0) {
    console.log("");
    console.log(`FAILURES (${failures.length}):`);
    for (const f of failures) console.log(`  - ${f}`);
    process.exit(1);
  }
  console.log("");
  console.log("Nav audit: ALL OK");
  process.exit(0);
}

main();
