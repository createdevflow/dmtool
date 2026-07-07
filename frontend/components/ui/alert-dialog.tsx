"use client";

import * as React from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogPortal,
  DialogOverlay,
  DialogClose,
} from "./dialog";
import { cn } from "@/lib/utils";

// AlertDialog is a thin wrapper over Dialog. Same Radix primitive
// under the hood — the only difference is a heavier overlay (the
// action is destructive, we want the user to fully commit) and a
// slightly tighter content width. Action buttons usually render
// "Cancel" on the left and the destructive action on the right.
//
// We re-export Dialog + the same subcomponents so consumers can
// import either set with the same names; both render identically.
const AlertDialog = Dialog;
const AlertDialogTrigger = DialogTrigger;
const AlertDialogPortal = DialogPortal;
const AlertDialogClose = DialogClose;

const AlertDialogOverlay = React.forwardRef<
  React.ElementRef<typeof DialogOverlay>,
  React.ComponentPropsWithoutRef<typeof DialogOverlay>
>(({ className, ...props }, ref) => (
  <DialogOverlay
    ref={ref}
    // Heavier dim than the standard Dialog so the destructive
    // choice reads as "stop and decide".
    className={cn("bg-black/55", className)}
    {...props}
  />
));
AlertDialogOverlay.displayName = "AlertDialogOverlay";

const AlertDialogContent = React.forwardRef<
  React.ElementRef<typeof DialogContent>,
  React.ComponentPropsWithoutRef<typeof DialogContent>
>(({ className, children, ...props }, ref) => (
  <DialogContent
    ref={ref}
    maxWidthClass="max-w-md"
    className={cn("gap-3", className)}
    {...props}
  >
    {children}
  </DialogContent>
));
AlertDialogContent.displayName = "AlertDialogContent";

const AlertDialogHeader = DialogHeader;
const AlertDialogFooter = DialogFooter;
const AlertDialogTitle = DialogTitle;
const AlertDialogDescription = DialogDescription;

export {
  AlertDialog,
  AlertDialogPortal,
  AlertDialogOverlay,
  AlertDialogTrigger,
  AlertDialogClose,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
};
