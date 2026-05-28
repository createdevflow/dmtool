"use client";

import Link from "next/link";
import { ArrowRight, Activity } from "lucide-react";
import { Button } from "@/components/ui/button";
import { motion } from "framer-motion";

export default function LandingPage() {
  return (
    <div className="bg-white text-slate-900 font-sans selection:bg-brand-500/10 min-h-screen flex flex-col">
      
      {/* Main Narrative */}
      <main className="flex-1 flex flex-col items-center justify-center px-6 py-20 text-center space-y-12 max-w-5xl mx-auto">

        
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
          className="space-y-6"
        >
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-slate-50 border border-slate-100 mb-4">
             <span className="w-1.5 h-1.5 rounded-full bg-brand-500 animate-pulse" />
             <span className="text-[10px] font-black uppercase tracking-widest text-slate-400">Next-Gen Messaging Engine</span>
          </div>
          
          <h1 className="text-6xl md:text-[7.5rem] font-black tracking-tighter leading-[0.85] text-slate-900">
            Growth. <br />
            <span className="text-slate-300">Autonomous.</span>
          </h1>
          
          <p className="text-xl md:text-2xl text-slate-500 font-medium max-w-2xl mx-auto leading-relaxed">
            The high-performance platform for individuals who demand precision in digital interaction. Built for speed, designed for the elite.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.5, duration: 1 }}
          className="flex flex-col sm:flex-row items-center gap-4 w-full max-w-md"
        >
          <Link href="/register" className="w-full">
            <Button className="w-full h-16 rounded-2xl bg-slate-900 text-white hover:bg-slate-800 font-bold text-xl shadow-2xl shadow-slate-900/20 group">
              Get Started <ArrowRight className="ml-2 w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </Button>
          </Link>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.8, duration: 1 }}
          className="grid grid-cols-2 md:grid-cols-4 gap-12 pt-20 w-full border-t border-slate-50"
        >
           {[
             { label: "Performance", value: "99.9%" },
             { label: "Latency", value: "<100ms" },
             { label: "Success Rate", value: "98.4%" },
             { label: "Autonomous", value: "24/7" }
           ].map((stat, i) => (
             <div key={i} className="space-y-1">
                <p className="text-2xl font-black text-slate-900">{stat.value}</p>
                <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">{stat.label}</p>
             </div>
           ))}
        </motion.div>
      </main>
    </div>
  );
}

