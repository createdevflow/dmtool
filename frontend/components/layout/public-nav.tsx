"use client";

import Link from "next/link";
import { Activity } from "lucide-react";
import { Button } from "@/components/ui/button";

export function PublicNav() {
  return (
    <nav className="fixed top-0 left-0 right-0 z-50 bg-white/80 backdrop-blur-xl border-b border-slate-50">
      <div className="max-w-7xl mx-auto px-8">
        <div className="flex justify-between items-center h-20">
          <Link href="/" className="flex items-center gap-2 text-slate-900 font-black text-xl tracking-tighter">
            <div className="bg-slate-900 rounded p-1 text-white">
              <Activity className="w-4 h-4" />
            </div>
            <span>DMTool</span>
          </Link>

          <div className="flex items-center gap-8">
            <Link href="/login" className="text-sm font-bold text-slate-500 hover:text-slate-900 transition-colors">Sign In</Link>
            <Link href="/register">
              <Button className="rounded-full bg-slate-900 text-white hover:bg-slate-800 px-6 font-bold shadow-lg shadow-slate-900/10">
                Join Now
              </Button>
            </Link>
          </div>
        </div>
      </div>
    </nav>
  );
}

