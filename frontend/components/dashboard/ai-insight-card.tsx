"use client";

import { motion } from "framer-motion";
import { Sparkles, Target, Zap, Clock, ArrowRight, Info } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

interface AIInsightCardProps {
  insight: any;
}

export function AIInsightCard({ insight }: AIInsightCardProps) {
  const colorMap: any = {
    brand: "blue",
    rose: "rose",
    amber: "amber",
    emerald: "emerald",
  };

  const themeColor = colorMap[insight.color] || "blue";

  return (
    <motion.div 
      initial={{ opacity: 0, x: -10 }} 
      animate={{ opacity: 1, x: 0 }} 
      className="group bg-white border border-slate-200/60 rounded-2xl hover:border-slate-300 hover:shadow-xl hover:shadow-slate-200/40 transition-all flex flex-col md:flex-row gap-8 p-8 relative overflow-hidden"
    >
      <div className={`absolute left-0 top-0 w-1 h-full bg-${themeColor}-500 opacity-80`} />
      
      <div className="flex-1 space-y-6">
        <div className="flex items-center gap-3">
          <Badge variant="outline" className={`rounded-md border-0 bg-${themeColor}-50 text-${themeColor}-600 text-[10px] uppercase font-bold tracking-wider px-2 py-0.5`}>
            {insight.type || "Observation"}
          </Badge>
          <span className="text-[10px] text-slate-400 font-semibold uppercase tracking-widest flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5" /> High Impact
          </span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="space-y-2">
            <h4 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 flex items-center gap-2">
              <Target className="w-3.5 h-3.5 text-slate-300" /> Analysis
            </h4>
            <h3 className="text-lg font-semibold text-slate-900 leading-tight">{insight.title}</h3>
          </div>

          <div className="space-y-2">
            <h4 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 flex items-center gap-2">
              <Info className="w-3.5 h-3.5 text-slate-300" /> Rationale
            </h4>
            <p className="text-sm text-slate-500 leading-relaxed font-medium">
              {insight.description}
            </p>
          </div>
        </div>
      </div>

      <div className="flex flex-col justify-end md:border-l border-slate-100 md:pl-8 min-w-[200px]">
        <Button className={`rounded-xl h-12 px-6 font-semibold flex items-center justify-between group/btn bg-slate-900 hover:bg-slate-800 text-white shadow-none transition-all`}>
          <span>{insight.actionText}</span>
          <ArrowRight className="w-4 h-4 group-hover/btn:translate-x-1 transition-transform ml-4" />
        </Button>
      </div>
    </motion.div>
  );
}
