import * as React from "react"
import { cn } from "@/lib/utils"

// Skeleton — placeholder block that pulses while real content loads.
// Use as a 1:1 stand-in for the loaded element so the layout doesn't
// shift. Multiple skeletons compose into a believable loading state
// (a card's title bar, table rows, etc).
function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      aria-hidden="true"
      className={cn("animate-pulse rounded-md bg-slate-200/70 dark:bg-slate-800/70", className)}
      {...props}
    />
  )
}

export { Skeleton }
