"use client";

import { CheckCircle2, ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { motion } from "framer-motion";

const tiers = [
  {
    name: "Standard",
    price: "$0",
    desc: "For those exploring the new frontier of digital interaction.",
    features: ["1 Active Node", "Stream Logic", "24h Retain"],
    featured: false,
  },
  {
    name: "Professional",
    price: "$29",
    desc: "The complete engine for high-performance growth teams.",
    features: ["Unlimited Nodes", "Autonomous Strategy", "Priority Infrastructure", "Global Scale"],
    featured: true,
  },
];

export default function PricingPage() {
  return (
    <div className="py-48 px-8 bg-white min-h-screen text-[#1d1d1f] font-sans">
      <div className="max-w-[1400px] mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-24 items-end mb-40">
           <div className="lg:col-span-7">
              <span className="text-sm font-bold text-[#0066cc] mb-6 block">Subscription</span>
              <h1 className="text-7xl sm:text-[9rem] font-bold tracking-tight leading-[0.8] text-[#1d1d1f]">
                 Simple <br />
                 <span className="text-gray-300">Scale.</span>
              </h1>
           </div>
           <div className="lg:col-span-4 lg:col-start-9 pb-4">
              <p className="text-xl text-gray-400 font-medium leading-relaxed">
                 Transparent access to the world's most sophisticated messaging engine. No hidden tiers, just raw power.
              </p>
           </div>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
          {tiers.map((tier, i) => (
            <motion.div
              key={tier.name}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.1, duration: 1, ease: [0.16, 1, 0.3, 1] }}
              className={`relative p-16 rounded-[4rem] flex flex-col justify-between min-h-[600px] transition-all ${tier.featured ? 'bg-[#1d1d1f] text-white shadow-3xl shadow-gray-200' : 'bg-[#fcfcfd] border border-gray-50'}`}
            >
              {tier.featured && (
                 <div className="absolute top-12 right-12 text-xs font-bold bg-blue-600 px-4 py-1.5 rounded-full">Recommended</div>
              )}
              
              <div>
                 <h2 className="text-4xl font-bold mb-6">{tier.name}</h2>
                 <p className={`text-lg font-medium mb-12 leading-relaxed ${tier.featured ? 'text-gray-400' : 'text-gray-500'}`}>
                    {tier.desc}
                 </p>
                 
                 <div className="flex items-baseline gap-4 mb-16">
                    <span className="text-8xl font-bold tracking-tight">{tier.price}</span>
                    <span className={`text-sm font-bold ${tier.featured ? 'text-gray-600' : 'text-gray-300'}`}>/ month</span>
                 </div>

                 <ul className="space-y-6">
                    {tier.features.map((f) => (
                      <li key={f} className="flex items-center gap-4 text-sm font-semibold">
                         <div className={`w-1.5 h-1.5 rounded-full ${tier.featured ? 'bg-blue-500' : 'bg-gray-200'}`} />
                         {f}
                      </li>
                    ))}
                 </ul>
              </div>

              <Link href="/register" className="mt-20">
                 <Button className={`w-full h-20 rounded-[2rem] font-bold text-xl tracking-tight transition-transform hover:scale-[0.98] ${tier.featured ? 'bg-blue-600 hover:bg-blue-500 text-white shadow-xl shadow-blue-500/20' : 'bg-white text-[#1d1d1f] border border-gray-100 hover:bg-gray-50'}`}>
                    {tier.featured ? 'Unlock All Nodes' : 'Start Experimenting'} <ArrowRight className="ml-2 w-6 h-6" />
                 </Button>
              </Link>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
}
