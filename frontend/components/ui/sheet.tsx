"use client";

import * as React from "react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

// Sheet — side-drawer modal. Same Radix Dialog primitive as the
// `Dialog` component, but the content is anchored to one edge of the
// viewport and slides in. Used for: mobile sidebar, admin detail
// panels, upgrade prompts.
//
// `side` defaults to "right" (the most common case for desktop
// drawers). Other values: "left", "top", "bottom".
const Sheet = DialogPrimitive.Root;
const SheetTrigger = DialogPrimitive.Trigger;
const SheetClose = DialogPrimitive.Close;
const SheetPortal = DialogPrimitive.Portal;

interface SheetContentProps
  extends React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> {
  side?: "right" | "left" | "top" | "bottom";
  // Tailwind width class when side is left/right. Defaults to the
  // standard sidebar width (264px) so this can stand in for the
  // mobile sidebar unchanged.
  widthClass?: string;
}

const SheetContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  SheetContentProps
>(({ className, side = "right", widthClass = "w-72 sm:w-80", children, ...props }, ref) => {
  // Positioning + slide-in animation keyframes per side. The entry
  // animation is keyed off the Radix data-state so it runs on mount;
  // exit on Radix's unmount (we use forwardRef + the same animation
  // hooks a built-in portal → body mount cycle would give us).
  const sideClasses: Record<NonNullable<SheetContentProps["side"]>, string> = {
    right:
      "inset-y-0 right-0 h-full data-[state=open]:animate-sheet-right-in data-[state=closed]:animate-sheet-right-out",
    left:
      "inset-y-0 left-0 h-full data-[state=open]:animate-sheet-left-in data-[state=closed]:animate-sheet-left-out",
    top: "inset-x-0 top-0 data-[state=open]:animate-sheet-top-in data-[state=closed]:animate-sheet-top-out",
    bottom:
      "inset-x-0 bottom-0 data-[state=open]:animate-sheet-bottom-in data-[state=closed]:animate-sheet-bottom-out",
  };
  const isHorizontal = side === "left" || side === "right";
  return (
    <SheetPortal>
      <DialogPrimitive.Overlay
        className={cn(
          "fixed inset-0 z-50 bg-black/40",
          "data-[state=open]:animate-overlay-in data-[state=closed]:animate-overlay-out"
        )}
      />
      <DialogPrimitive.Content
        ref={ref}
        className={cn(
          "fixed z-50 bg-white shadow-2xl flex flex-col gap-4 p-6",
          isHorizontal ? `h-full ${widthClass}` : "w-full",
          sideClasses[side],
          className
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close
          aria-label="Close"
          className="absolute right-4 top-4 rounded-lg p-1 text-slate-400 hover:text-slate-900 hover:bg-slate-100 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-300"
        >
          <X className="h-4 w-4" />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </SheetPortal>
  );
});
SheetContent.displayName = "SheetContent";

const SheetHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div className={cn("flex flex-col space-y-1 text-left", className)} {...props} />
);
SheetHeader.displayName = "SheetHeader";

const SheetTitle = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Title>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title
    ref={ref}
    className={cn("text-lg font-semibold leading-none tracking-tight text-slate-900", className)}
    {...props}
  />
));
SheetTitle.displayName = "SheetTitle";

const SheetDescription = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Description>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description
    ref={ref}
    className={cn("text-sm text-slate-500", className)}
    {...props}
  />
));
SheetDescription.displayName = "SheetDescription";

export {
  Sheet,
  SheetPortal,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
};
