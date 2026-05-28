"use client";

import Link from "next/link";
import { ArrowRight, Book, Code2, Sparkles, LineChart } from "lucide-react";
import { Input } from "@/components/ui/input";

const links = [
  { name: "Getting Started", icon: Book, href: "#" },
  { name: "Connecting Accounts", icon: Code2, href: "#" },
  { name: "Using SEO Tools", icon: LineChart, href: "#" },
  { name: "AI Features", icon: Sparkles, href: "#" },
];

export default function DocsPage() {
  return (
    <div className="flex w-full max-w-7xl mx-auto px-6 lg:px-8 py-10 min-h-[calc(100vh-64px)]">
      {/* Docs Sidebar */}
      <aside className="w-64 shrink-0 hidden md:block pt-6 border-r border-border pr-6">
        <h3 className="font-semibold text-lg mb-4">Documentation</h3>
        <nav className="flex flex-col gap-2">
           {links.map((link) => (
             <Link key={link.name} href={link.href} className="flex items-center gap-3 text-sm text-slate-600 hover:text-foreground hover:bg-slate-50 dark:hover:bg-slate-800/50 p-2 rounded-lg transition-colors">
                <link.icon className="w-4 h-4 text-slate-400" />
                {link.name}
             </Link>
           ))}
        </nav>
      </aside>
      
      {/* Docs Content */}
      <main className="flex-1 md:pl-10 pt-6">
         <div className="max-w-3xl">
            <h1 className="text-4xl font-bold tracking-tight mb-4">Introduction to DMTool</h1>
            <p className="text-lg text-slate-500 mb-8">Learn how to extract insights, train models, and manage bulk campaigns using our developer-ready marketing platform.</p>
            
            <div className="mb-10 w-full max-w-md">
               <Input placeholder="Search documentation..." className="shadow-sm" />
            </div>

            <div className="prose prose-slate dark:prose-invert">
              <h3 className="text-xl font-semibold mb-4 mt-8">Quick Navigation</h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 not-prose">
                <a href="#" className="group relative rounded-2xl border border-border bg-card p-6 hover:shadow-md transition-shadow">
                  <h4 className="font-semibold text-foreground mb-2 flex items-center justify-between">
                    Authentication <ArrowRight className="w-4 h-4 text-slate-400 group-hover:translate-x-1 group-hover:text-brand-600 transition-all" />
                  </h4>
                  <p className="text-sm text-slate-500">Secure your API calls using Bearer Tokens.</p>
                </a>
                <a href="#" className="group relative rounded-2xl border border-border bg-card p-6 hover:shadow-md transition-shadow">
                  <h4 className="font-semibold text-foreground mb-2 flex items-center justify-between">
                    Crawler Webhooks <ArrowRight className="w-4 h-4 text-slate-400 group-hover:translate-x-1 group-hover:text-brand-600 transition-all" />
                  </h4>
                  <p className="text-sm text-slate-500">Subscribe natively to deep SEO scanning events.</p>
                </a>
              </div>
            </div>
         </div>
      </main>
    </div>
  );
}
