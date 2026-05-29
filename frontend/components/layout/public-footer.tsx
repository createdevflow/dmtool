"use client";

import Link from "next/link";

export function PublicFooter() {
  return (
    <footer className="py-12 border-t border-slate-50 px-8">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row justify-between items-center gap-6">
        <p className="text-xs font-bold text-slate-300 tracking-tight">© 2026 DMTool Intelligence. Designed for the Bold.</p>
        <div className="flex gap-8 text-[10px] font-black text-slate-400 uppercase tracking-widest">
           <Link href="/privacy" className="hover:text-slate-900 transition-colors">Privacy</Link>
           <Link href="#" className="hover:text-slate-900 transition-colors">Terms</Link>
           <Link href="#" className="hover:text-slate-900 transition-colors">Twitter</Link>
        </div>
      </div>
    </footer>
  );
}
