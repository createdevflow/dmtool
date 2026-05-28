"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Activity } from "lucide-react";
import { cn } from "@/lib/utils";

export function Navbar() {
  const pathname = usePathname();
  
  return (
    <header className="fixed top-0 left-0 right-0 z-50 glass-panel h-16 flex items-center">
      <div className="container mx-auto px-6 max-w-7xl flex items-center justify-between">
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-2 text-foreground font-semibold text-lg group">
            <div className="bg-brand-600 rounded-lg p-1.5 text-white group-hover:bg-brand-500 transition-colors">
              <Activity className="w-5 h-5" />
            </div>
            <span>DMTool</span>
          </Link>
          
          <nav className="hidden md:flex items-center gap-6 text-sm font-medium text-slate-500 dark:text-slate-400">
            <Link href="/" className={cn("hover:text-foreground transition-colors", pathname === "/" && "text-foreground")}>Home</Link>
            <Link href="/pricing" className={cn("hover:text-foreground transition-colors", pathname === "/pricing" && "text-foreground")}>Pricing</Link>
            <Link href="/docs" className={cn("hover:text-foreground transition-colors", pathname === "/docs" && "text-foreground")}>Documentation</Link>
          </nav>
        </div>
        
        <div className="flex items-center gap-4">
          <Link href="/login" className="text-sm font-medium hover:text-foreground transition-colors text-slate-500 dark:text-slate-400 hidden sm:block">
            Log in
          </Link>
          <Link href="/register" className="text-sm font-medium bg-foreground text-background px-4 py-2 rounded-full hover:opacity-90 transition-opacity">
            Sign up
          </Link>
        </div>
      </div>
    </header>
  );
}
