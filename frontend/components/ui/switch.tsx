"use client"

import * as React from "react"
import { cn } from "@/lib/utils"

interface SwitchProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const Switch = React.forwardRef<HTMLInputElement, SwitchProps>(
  ({ className, ...props }, ref) => {
    return (
      <div className={cn("inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 relative", 
        props.checked || props.defaultChecked ? "bg-brand-600" : "bg-slate-200 dark:bg-slate-800",
        className)}>
        <input
          type="checkbox"
          ref={ref}
          className="sr-only"
          {...props}
        />
        <div
          className={cn(
            "pointer-events-none block h-5 w-5 rounded-full bg-white shadow-lg ring-0 transition-transform",
            props.checked || props.defaultChecked ? "translate-x-5" : "translate-x-0"
          )}
        />
      </div>
    )
  }
)
Switch.displayName = "Switch"

export { Switch }
