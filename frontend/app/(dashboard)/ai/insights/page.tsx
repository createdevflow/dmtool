"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  BrainCircuit, TrendingUp, Search, Filter, 
  ArrowRight, Sparkles, Zap, ChevronRight,
  TrendingDown, Info, AlertTriangle, CheckCircle2,
  Loader2
} from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { Card, CardContent } from "@/components/ui/card";

export default function AIInsightsPage() {
  const [insights, setInsights] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("all");

  useEffect(() => {
    const fetchInsights = async () => {
      try {
        const pRes = await dashboardApi.getProjects();
        const allProjects = pRes.data.data || [];
        if (allProjects.length > 0) {
          const selected = allProjects[allProjects.length - 1];
          const res = await dashboardApi.getInsights(selected.id);
          setInsights(res.data.data ?? []);
        }
      } catch (err) {
        console.error("Failed to fetch insights", err);
      } finally {
        setLoading(false);
      }
    };
    fetchInsights();

  }, []);

  const filteredInsights = insights.filter(i => filter === "all" || i.type === filter);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-6xl mx-auto pb-10">
      
      {/* Header section with Stats */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}>
          <div className="flex items-center gap-3 mb-2">
            <div className="bg-brand-600 rounded-lg p-2 text-white shadow-lg shadow-brand-500/20">
              <BrainCircuit className="w-6 h-6" />
            </div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground">AI Intelligence</h1>
          </div>
          <p className="text-slate-500">Autonomous strategy generated from your real-time performance data.</p>
        </motion.div>

        <div className="flex items-center gap-3">
          <div className="flex bg-slate-100 dark:bg-slate-800 rounded-xl p-1.5 border border-border overflow-hidden">
            {["all", "CRITICAL", "OPPORTUNITY", "WARNING"].map((f) => (
              <button 
                key={f}
                onClick={() => setFilter(f)}
                className={`px-4 py-1.5 rounded-lg text-xs font-bold transition-all uppercase tracking-wider ${
                   filter === f 
                   ? "bg-white dark:bg-slate-700 shadow-sm text-foreground" 
                   : "text-slate-500 hover:text-slate-900 dark:hover:text-white"
                }`}
              >
                {f}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Feature Highlight Card */}
      <motion.div 
        initial={{ opacity: 0, y: 20 }} 
        animate={{ opacity: 1, y: 0 }}
        className="bg-gradient-to-r from-brand-600 to-indigo-600 rounded-3xl p-8 text-white relative overflow-hidden shadow-2xl"
      >
        <div className="absolute top-0 right-0 w-64 h-64 bg-white/10 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2" />
        <div className="relative z-10 flex flex-col md:flex-row md:items-center gap-8">
           <div className="bg-white/20 backdrop-blur-xl p-4 rounded-2xl border border-white/20">
              <Sparkles className="w-10 h-10 text-white" />
           </div>
           <div>
              <h2 className="text-2xl font-bold mb-2">Next-Gen Prediction Engine</h2>
              <p className="text-brand-100 max-w-xl font-medium">Your current growth trajectory suggests a 22% increase in visibility if you execute the "Critical" tasks within the next 48 hours.</p>
           </div>
           <button className="md:ml-auto bg-white text-brand-700 hover:bg-brand-50 font-bold px-6 py-3 rounded-xl transition-all shadow-xl shadow-brand-900/20 active:scale-95">
              Generate New Strategy
           </button>
        </div>
      </motion.div>

      {/* Insights Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <AnimatePresence mode="popLayout">
          {filteredInsights.map((insight, i) => (
            <motion.div 
              key={insight.id} 
              layout
              initial={{ opacity: 0, scale: 0.9 }} 
              animate={{ opacity: 1, scale: 1 }} 
              exit={{ opacity: 0, scale: 0.9 }}
              transition={{ delay: i * 0.05 }}
            >
              <Card className="h-full border border-border hover:border-brand-500 transition-all group overflow-hidden relative">
                <div className={`absolute top-0 left-0 w-full h-1 bg-${insight.color === 'rose' ? 'rose' : insight.color === 'brand' ? 'brand' : 'amber'}-500/50`} />
                <CardContent className="p-6">
                  <div className="flex justify-between items-start mb-6">
                    <div className={`p-2 rounded-lg bg-${insight.color === 'rose' ? 'rose' : insight.color === 'brand' ? 'brand' : 'amber'}-500/10 text-${insight.color === 'rose' ? 'rose' : insight.color === 'brand' ? 'brand' : 'amber'}-500`}>
                       {insight.type === 'CRITICAL' ? <AlertTriangle className="w-5 h-5" /> : insight.type === 'OPPORTUNITY' ? <Zap className="w-5 h-5" /> : <Info className="w-5 h-5" />}
                    </div>
                    <span className={`text-[10px] font-black uppercase tracking-widest px-2 py-0.5 rounded bg-${insight.color === 'rose' ? 'rose' : insight.color === 'brand' ? 'brand' : 'amber'}-500/10 text-${insight.color === 'rose' ? 'rose' : insight.color === 'brand' ? 'brand' : 'amber'}-500`}>
                      {insight.type}
                    </span>
                  </div>

                  <h3 className="text-lg font-bold mb-3 group-hover:text-brand-600 transition-colors">{insight.title}</h3>
                  <p className="text-sm text-slate-500 mb-8 leading-relaxed font-medium">{insight.description}</p>
                  
                  <div className="flex items-center justify-between pt-4 border-t border-border">
                     <span className="text-xs font-bold text-slate-400">Impact: High</span>
                     <button className="flex items-center gap-1.5 text-sm font-bold text-brand-600 hover:gap-3 transition-all">
                        {insight.actionText} <ArrowRight className="w-4 h-4" />
                     </button>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      {filteredInsights.length === 0 && (
         <div className="text-center py-20 bg-slate-50 dark:bg-white/5 rounded-3xl border border-dashed border-border">
            <p className="text-slate-500 font-bold">No insights found for this filter.</p>
         </div>
      )}

    </div>
  );
}
