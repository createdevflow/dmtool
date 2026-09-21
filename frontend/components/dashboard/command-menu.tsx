"use client";

// CommandMenu — global keyboard launcher (Cmd/Ctrl+K).
//
// Phase 8 rewrite. The previous version had a 4-item hardcoded list
// and pointed at several dead routes (`/social-insights`,
// `/content-ai`, `/settings`). The fix:
//
//   1. Items are derived from the SAME `navGroups` data the desktop
//      and mobile sidebars consume (frontend/components/dashboard/
//      sidebar.tsx). Adding a route to the sidebar is now
//      automatically a route in command-palette. The single source
//      of truth lives next to the rest of the nav config.
//   2. We honour the user's current dashboard mode. A user in
//      "social" mode won't see SEO Intelligence items in the
//      palette; an item hidden by `modeExtras` is also hidden
//      here. Switching modes is one click on the sidebar
//      mode-switcher.
//   3. The dialog is wrapped in our shared Dialog primitive so
//      Escape closes, click-outside closes, focus is trapped, and
//      screen-reader semantics carry through.
//   4. Stale `/settings` link removed in favour of
//      `/projects/settings` (the only Settings page that exists).
//
// Stale-route check passes by construction now; the phase8_verify
// tool audits `navGroups[]` against the actual page tree on disk.

import * as React from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDashboardMode } from "./dashboard-mode-context";
import { navGroups, type NavItem } from "./sidebar";

export function CommandMenu() {
  const [open, setOpen] = React.useState(false);
  const router = useRouter();
  const { mode } = useDashboardMode();

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const onSelect = React.useCallback(
    (href: string) => () => {
      setOpen(false);
      router.push(href);
    },
    [router]
  );

  // Mode-aware group list. The same two-level filter the Sidebar
  // applies: group's modes must include the active mode, and an
  // item with modeExtras must list the active mode (otherwise it's
  // a drill-down visible only inside the parent group's home mode).
  const visibleGroups = React.useMemo(
    () =>
      navGroups
        .filter((g) => g.modes.includes(mode))
        .map((g) => ({
          ...g,
          items: g.items.filter(
            (item) =>
              item.modeExtras === undefined || item.modeExtras.includes(mode)
          ),
        }))
        .filter((g) => g.items.length > 0),
    [mode]
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent
        // The cmdk shell handles its own padding/width; we drop
        // the default Dialog padding/width and give it a tighter
        // command-palette shape.
        className="max-w-xl gap-0 p-0 overflow-hidden"
        aria-label="Command palette"
        hideClose
      >
        <DialogTitle className="sr-only">Command palette</DialogTitle>
        <Command label="Global command palette" className="flex h-full w-full flex-col bg-transparent">
          <div className="flex items-center border-b border-slate-100 px-3">
            <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
            <Command.Input
              autoFocus
              className="flex h-12 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-slate-400"
              placeholder="Type a command or jump to a page…"
            />
          </div>
          <Command.List className="max-h-[60vh] overflow-y-auto overflow-x-hidden p-2 subtle-scrollbar">
            <Command.Empty className="py-6 text-center text-sm text-slate-500">
              No matches. Try a page name like “Billing”.
            </Command.Empty>

            {visibleGroups.map((group) => (
              <Command.Group
                key={group.title}
                heading={group.title}
                className="px-1 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-400"
              >
                {group.items.map((item) => (
                  <PaletteItem
                    key={item.href}
                    item={item}
                    onSelect={onSelect(item.href)}
                  />
                ))}
              </Command.Group>
            ))}
          </Command.List>
        </Command>
      </DialogContent>
    </Dialog>
  );
}

// PaletteItem — renders one row. We split into a sub-component so
// the type carries the icon through correctly (palette items need
// the NavItem's icon type, not the lucide component shape).
function PaletteItem({
  item,
  onSelect,
}: {
  item: NavItem;
  onSelect: () => void;
}) {
  const Icon = item.icon;
  return (
    <Command.Item
      onSelect={onSelect}
      value={`${item.name} ${item.href}`}
      className="flex cursor-pointer select-none items-center gap-2 rounded-lg px-3 py-2.5 text-sm outline-none aria-selected:bg-brand-50 aria-selected:text-brand-700 hover:bg-slate-50"
    >
      <Icon className="h-4 w-4 text-slate-400 shrink-0" />
      <span className="flex-1 truncate font-medium">{item.name}</span>
      <span className="text-[10px] font-mono text-slate-400 truncate">{item.href}</span>
    </Command.Item>
  );
}
