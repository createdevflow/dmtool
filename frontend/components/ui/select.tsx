import * as React from "react"
import { cn } from "@/lib/utils"

const Select = React.forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement> & { onValueChange?: (value: string) => void }>(
  ({ className, onValueChange, onChange, ...props }, ref) => {
    return (
      <select
        ref={ref}
        className={cn(
          "flex h-10 w-full items-center justify-between rounded-md border border-slate-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-950 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-800 dark:bg-slate-950 dark:ring-offset-slate-950 dark:placeholder:text-slate-400 dark:focus:ring-slate-300",
          className
        )}
        onChange={(e) => {
          if (onChange) onChange(e);
          if (onValueChange) onValueChange(e.target.value);
        }}
        {...props}
      />
    )
  }
)
Select.displayName = "Select"

const SelectGroup = React.forwardRef<HTMLOptGroupElement, React.OptgroupHTMLAttributes<HTMLOptGroupElement>>(
  ({ className, ...props }, ref) => (
    <optgroup ref={ref} className={cn("", className)} {...props} />
  )
)
SelectGroup.displayName = "SelectGroup"

const SelectValue = React.forwardRef<HTMLSpanElement, React.HTMLAttributes<HTMLSpanElement>>(
  ({ className, ...props }, ref) => (
    <span ref={ref} className={cn("hidden", className)} {...props} />
  )
)
SelectValue.displayName = "SelectValue"

const SelectTrigger = React.forwardRef<HTMLButtonElement, React.ButtonHTMLAttributes<HTMLButtonElement>>(
  ({ className, children, ...props }, ref) => (
    <div className={cn("relative flex w-full items-center", className)} {...(props as any)}>
      {children}
    </div>
  )
)
SelectTrigger.displayName = "SelectTrigger"

const SelectContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, children, ...props }, ref) => (
    <React.Fragment>{children}</React.Fragment>
  )
)
SelectContent.displayName = "SelectContent"

const SelectLabel = React.forwardRef<HTMLOptionElement, React.OptionHTMLAttributes<HTMLOptionElement>>(
  ({ className, ...props }, ref) => (
    <option ref={ref} disabled className={cn("font-semibold", className)} {...props} />
  )
)
SelectLabel.displayName = "SelectLabel"

const SelectItem = React.forwardRef<HTMLOptionElement, React.OptionHTMLAttributes<HTMLOptionElement>>(
  ({ className, children, ...props }, ref) => (
    <option ref={ref} className={cn("", className)} {...props}>
      {children}
    </option>
  )
)
SelectItem.displayName = "SelectItem"

const SelectSeparator = React.forwardRef<HTMLHRElement, React.HTMLAttributes<HTMLHRElement>>(
  ({ className, ...props }, ref) => (
    <hr ref={ref} className={cn("-mx-1 my-1 h-px bg-slate-100 dark:bg-slate-800", className)} {...props} />
  )
)
SelectSeparator.displayName = "SelectSeparator"

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
}
