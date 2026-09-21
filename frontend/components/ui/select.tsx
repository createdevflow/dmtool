import * as React from "react";
import { cn } from "@/lib/utils";

type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement> & {
  onValueChange?: (value: string) => void;
};

function elementName(child: React.ReactElement): string {
  const t = child.type as { displayName?: string };
  return t?.displayName ?? "";
}

function collectSelectItems(children: React.ReactNode): React.ReactElement[] {
  const out: React.ReactElement[] = [];
  React.Children.forEach(children, (child) => {
    if (!React.isValidElement(child)) return;
    const name = elementName(child);
    if (name === "SelectItem" || child.type === "option") {
      out.push(child);
      return;
    }
    const nested = (child.props as { children?: React.ReactNode }).children;
    if (nested != null) {
      out.push(...collectSelectItems(nested));
    }
  });
  return out;
}

function collectTriggerClass(children: React.ReactNode): string {
  let cls = "";
  React.Children.forEach(children, (child) => {
    if (!React.isValidElement(child)) return;
    if (elementName(child) === "SelectTrigger") {
      cls = String((child.props as { className?: string }).className ?? "");
    }
    const nested = (child.props as { children?: React.ReactNode }).children;
    if (nested != null) {
      const inner = collectTriggerClass(nested);
      if (inner) cls = inner;
    }
  });
  return cls;
}

const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, onValueChange, onChange, children, ...props }, ref) => {
    const items = collectSelectItems(children);
    const triggerClass = collectTriggerClass(children);
    return (
      <select
        ref={ref}
        className={cn(
          "flex h-10 w-full items-center justify-between rounded-md border border-slate-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-950 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-800 dark:bg-slate-950 dark:ring-offset-slate-950 dark:placeholder:text-slate-400 dark:focus:ring-slate-300",
          triggerClass,
          className
        )}
        onChange={(e) => {
          onChange?.(e);
          onValueChange?.(e.target.value);
        }}
        {...props}
      >
        {items}
      </select>
    );
  }
);
Select.displayName = "Select";

const SelectGroup = React.forwardRef<
  HTMLOptGroupElement,
  React.OptgroupHTMLAttributes<HTMLOptGroupElement>
>(({ className, ...props }, ref) => (
  <optgroup ref={ref} className={cn("", className)} {...props} />
));
SelectGroup.displayName = "SelectGroup";

const SelectValue = React.forwardRef<HTMLSpanElement, React.HTMLAttributes<HTMLSpanElement>>(
  (_props, _ref) => null
);
SelectValue.displayName = "SelectValue";

const SelectTrigger = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  (_props, _ref) => null
);
SelectTrigger.displayName = "SelectTrigger";

const SelectContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ children }, _ref) => <>{children}</>
);
SelectContent.displayName = "SelectContent";

const SelectLabel = React.forwardRef<
  HTMLOptionElement,
  React.OptionHTMLAttributes<HTMLOptionElement>
>(({ className, ...props }, ref) => (
  <option ref={ref} disabled className={cn("font-semibold", className)} {...props} />
));
SelectLabel.displayName = "SelectLabel";

const SelectItem = React.forwardRef<
  HTMLOptionElement,
  React.OptionHTMLAttributes<HTMLOptionElement>
>(({ className, children, ...props }, ref) => (
  <option ref={ref} className={cn("", className)} {...props}>
    {children}
  </option>
));
SelectItem.displayName = "SelectItem";

const SelectSeparator = React.forwardRef<HTMLHRElement, React.HTMLAttributes<HTMLHRElement>>(
  (_props, _ref) => null
);
SelectSeparator.displayName = "SelectSeparator";

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
};
