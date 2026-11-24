"use client";

// CommandMenu — global keyboard launcher (Cmd/Ctrl+K).
//
// Items come from the same `navGroups` data the two-pane sidebar uses.
// Adding a route to the sidebar adds it to the palette. Admin-only
// rails stay hidden unless the user cookie role is admin. Dashboard
// Search/Social/Combined mode does not filter this list.

import * as React from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { navGroups, type NavItem } from "./sidebar";
import { readCookie, COOKIE_USER } from "@/lib/auth-cookie";

function isAdminUser(): boolean {
  if (typeof document === "undefined") return false;
  const raw = readCookie(COOKIE_USER);
  if (!raw) return false;
  try {
    return JSON.parse(raw)?.role === "admin";
  } catch {
    return false;
  }
}

export function CommandMenu() {
  const [open, setOpen] = React.useState(false);
  const [admin, setAdmin] = React.useState(false);
  const router = useRouter();

  React.useEffect(() => {
    setAdmin(isAdminUser());
  }, []);

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

  const visibleGroups = React.useMemo(
    () =>
      navGroups
        .filter((g) => !g.adminOnly || admin)
        .map((g) => ({
          title: g.title,
          icon: g.icon,
          items: g.sections.flatMap((s) => s.items),
        }))
        .filter((g) => g.items.length > 0),
    [admin]
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent
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
                    FallbackIcon={group.icon}
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

function PaletteItem({
  item,
  onSelect,
  FallbackIcon,
}: {
  item: NavItem;
  onSelect: () => void;
  FallbackIcon: NonNullable<NavItem["icon"]>;
}) {
  const Icon = item.icon ?? FallbackIcon;
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
