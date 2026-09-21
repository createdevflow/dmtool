"use client";

import * as React from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { cn } from "@/lib/utils";

// Tooltip — hover-label primitive. Built on Radix Tooltip so it
// inherits show-with-delay, safe-area positioning, and proper
// aria-describedby wiring out of the box.
//
// Usage:
//   <Tooltip>
//     <TooltipTrigger asChild><Button>…</Button></TooltipTrigger>
//     <TooltipContent side="top">Edit profile</TooltipContent>
//   </Tooltip>
//
// Wrap your app in <TooltipProvider> once (we do this in the dashboard
// layout) so the delay / skip-delay timings are global rather than
// per-instance.

const TooltipProvider = TooltipPrimitive.Provider;

const Tooltip = TooltipPrimitive.Root;

const TooltipTrigger = TooltipPrimitive.Trigger;

const TooltipContent = React.forwardRef<
  React.ElementRef<typeof TooltipPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content>
>(({ className, sideOffset = 6, ...props }, ref) => (
  <TooltipPrimitive.Portal>
    <TooltipPrimitive.Content
      ref={ref}
      sideOffset={sideOffset}
      className={cn(
        "z-50 overflow-hidden rounded-md bg-slate-900 px-2.5 py-1 text-xs font-medium text-white shadow-md",
        "data-[state=delayed-open]:animate-tooltip-in",
        "data-[state=closed]:animate-tooltip-out",
        className
      )}
      {...props}
    />
  </TooltipPrimitive.Portal>
));
TooltipContent.displayName = TooltipPrimitive.Content.displayName;

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
