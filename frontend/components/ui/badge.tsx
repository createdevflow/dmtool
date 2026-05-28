import * as React from "react"
import { cn } from "@/lib/utils"

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "default" | "secondary" | "destructive" | "outline"
}

function Badge({ className, variant = "default", ...props }: BadgeProps) {
  return (
    <div
      className={cn(
        "inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
        {
          "border-transparent bg-foreground text-background shadow hover:bg-foreground/80":
            variant === "default",
          "border-transparent bg-slate-100 text-slate-900 hover:bg-slate-100/80 dark:bg-slate-800 dark:text-slate-50 dark:hover:bg-slate-800/80":
            variant === "secondary",
          "border-transparent bg-red-500 text-slate-50 shadow hover:bg-red-500/80":
            variant === "destructive",
          "text-slate-950 dark:text-slate-50": variant === "outline",
        },
        className
      )}
      {...props}
    />
  )
}

export { Badge }
export default Badge
