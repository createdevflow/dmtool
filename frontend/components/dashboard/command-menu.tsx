"use client";

import * as React from "react"
import { useRouter } from "next/navigation"
import { Command } from "cmdk"
import { Search, Map, BarChart3, Settings, PenTool } from "lucide-react"

export function CommandMenu() {
  const [open, setOpen] = React.useState(false)
  const router = useRouter()

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((open) => !open)
      }
    }

    document.addEventListener("keydown", down)
    return () => document.removeEventListener("keydown", down)
  }, [])

  const runCommand = React.useCallback((command: () => unknown) => {
    setOpen(false)
    command()
  }, [])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[100] bg-background/80 backdrop-blur-sm px-4 pt-[20vh]" onClick={() => setOpen(false)}>
      <div 
        className="mx-auto max-w-xl overflow-hidden rounded-xl border border-border bg-background shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <Command className="flex h-full w-full flex-col overflow-hidden bg-transparent">
          <div className="flex items-center border-b border-border px-3">
             <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
             <Command.Input 
                autoFocus 
                className="flex h-12 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50" 
                placeholder="Analyze competitor, generate content, or jump to..." 
             />
          </div>
          <Command.List className="max-h-[300px] overflow-y-auto overflow-x-hidden p-2 subtle-scrollbar">
            <Command.Empty className="py-6 text-center text-sm">No results found.</Command.Empty>
            
            <Command.Group heading="Suggest Actions" className="px-2 py-1.5 text-xs font-semibold text-muted-foreground text-slate-500">
              <Command.Item 
                onSelect={() => runCommand(() => router.push('/dashboard'))}
                className="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2.5 text-sm outline-none aria-selected:bg-accent aria-selected:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                <Search className="mr-2 h-4 w-4 text-brand-500" />
                Analyze a Competitor
              </Command.Item>
              <Command.Item 
                onSelect={() => runCommand(() => router.push('/content-ai'))}
                className="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2.5 text-sm outline-none aria-selected:bg-accent aria-selected:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                <PenTool className="mr-2 h-4 w-4 text-brand-500" />
                Generate New Content
              </Command.Item>
            </Command.Group>

            <Command.Group heading="Navigation" className="px-2 py-1.5 text-xs font-semibold text-muted-foreground text-slate-500 mt-2">
              <Command.Item 
                onSelect={() => runCommand(() => router.push('/social-insights'))}
                className="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2.5 text-sm outline-none aria-selected:bg-accent aria-selected:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                <BarChart3 className="mr-2 h-4 w-4" />
                Social Insights
              </Command.Item>
              <Command.Item 
                onSelect={() => runCommand(() => router.push('/dashboard'))}
                className="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2.5 text-sm outline-none aria-selected:bg-accent aria-selected:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                <Settings className="mr-2 h-4 w-4" />
                Settings
              </Command.Item>
            </Command.Group>
            
          </Command.List>
        </Command>
      </div>
    </div>
  )
}
