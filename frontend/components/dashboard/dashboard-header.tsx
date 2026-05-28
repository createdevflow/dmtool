"use client";

import { 
  Globe, Instagram, Twitter, Linkedin, Facebook, 
  Plus, Calendar, ChevronDown, LayoutGrid
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { useState } from "react";
import { useRouter } from "next/navigation";

interface DashboardHeaderProps {
  project: any;
  projects?: any[];
  onProjectChange?: (project: any) => void;
  onAddSource: () => void;
  onDateRangeChange?: (days: number) => void;
}

const DATE_RANGES = [
  { label: "Last 7 Days", days: 7 },
  { label: "Last 14 Days", days: 14 },
  { label: "Last 30 Days", days: 30 },
  { label: "Last 90 Days", days: 90 },
];

export function DashboardHeader({ project, projects = [], onProjectChange, onAddSource, onDateRangeChange }: DashboardHeaderProps) {
  const router = useRouter();
  const [selectedRange, setSelectedRange] = useState(DATE_RANGES[2]);

  const sources = [
    { name: "Website", connected: !!project?.url, icon: Globe },
    { name: "Instagram", connected: !!project?.ig_handle, icon: Instagram },
    { name: "Twitter", connected: !!project?.twitter_handle, icon: Twitter },
    { name: "LinkedIn", connected: !!project?.linkedin_handle, icon: Linkedin },
    { name: "Facebook", connected: !!project?.facebook_handle, icon: Facebook },
  ];

  const handleRangeSelect = (range: typeof DATE_RANGES[0]) => {
    setSelectedRange(range);
    onDateRangeChange?.(range.days);
  };

  return (
    <div className="flex flex-col gap-8 pb-8 border-b border-slate-100">
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6">
        {/* Project Switcher & Name */}
        <div className="flex items-center gap-4">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="p-0 hover:bg-transparent flex items-center gap-3 group">
                <div className="w-11 h-11 rounded-xl bg-slate-900 flex items-center justify-center text-white transition-all group-hover:bg-brand-600">
                  <LayoutGrid className="w-5 h-5" />
                </div>
                <div className="text-left">
                  <div className="flex items-center gap-1.5">
                    <h1 className="text-lg font-semibold tracking-tight text-slate-900">{project?.name || "Select Project"}</h1>
                    <ChevronDown className="w-4 h-4 text-slate-400 group-hover:text-slate-600 transition-colors" />
                  </div>
                  <p className="text-xs font-medium text-slate-400">{project?.url || "No property connected"}</p>
                </div>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-64 rounded-xl shadow-xl border-slate-200">
              <DropdownMenuLabel className="text-[10px] font-semibold uppercase tracking-wider text-slate-400 px-3 py-2">Marketing Properties</DropdownMenuLabel>
              <div className="max-h-[300px] overflow-y-auto subtle-scrollbar">
                {projects.map((p) => (
                  <DropdownMenuItem 
                    key={p.id} 
                    onClick={() => onProjectChange?.(p)}
                    className="rounded-lg font-medium px-3 py-2.5 cursor-pointer"
                  >
                    <div className="flex items-center gap-2.5 w-full">
                      <div className={`w-2 h-2 rounded-full ${p.id === project?.id ? 'bg-emerald-500' : 'bg-slate-300'}`} />
                      {p.name}
                    </div>
                  </DropdownMenuItem>
                ))}
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="rounded-lg font-medium px-3 py-2.5 cursor-pointer text-brand-600 focus:text-brand-600" onClick={() => router.push('/projects/create')}>
                <Plus className="w-4 h-4 mr-2" /> Connect New Project
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Sources & Actions */}
        <div className="flex flex-wrap items-center gap-4">
          {/* Connection Status */}
          <div className="flex items-center bg-slate-50 p-1 rounded-xl border border-slate-200/60 gap-1 pr-3">
            {sources.map((source) => (
              <div 
                key={source.name} 
                className={`flex items-center gap-2 px-2.5 py-1.5 rounded-lg transition-all ${source.connected ? 'bg-white shadow-sm ring-1 ring-slate-200/50' : 'opacity-30'}`}
                title={source.connected ? `${source.name} Connected` : `${source.name} Disconnected`}
              >
                <source.icon className={`w-3.5 h-3.5 ${source.connected ? 'text-slate-900' : 'text-slate-400'}`} />
                <span className={`text-[10px] font-bold uppercase tracking-tight ${source.connected ? 'text-slate-900' : 'text-slate-400'}`}>
                  {source.connected ? 'Active' : ''}
                </span>
              </div>
            ))}
          </div>

          <div className="h-6 w-px bg-slate-200 mx-2 hidden sm:block" />

          {/* Date Range Selector — Functional */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" className="rounded-xl border-slate-200 font-medium gap-2 text-xs h-10 px-4 hover:bg-slate-50 transition-colors">
                <Calendar className="w-3.5 h-3.5 text-slate-500" />
                {selectedRange.label}
                <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44 rounded-xl shadow-lg border-slate-100 p-1">
              {DATE_RANGES.map((range) => (
                <DropdownMenuItem
                  key={range.days}
                  onClick={() => handleRangeSelect(range)}
                  className={`rounded-lg cursor-pointer text-sm font-medium px-3 py-2 ${selectedRange.days === range.days ? 'bg-slate-100 text-slate-900 font-semibold' : 'text-slate-600'}`}
                >
                  {range.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>

          {/* Add Data Source — routes to integrations */}
          <Button onClick={() => router.push('/integrations')} className="rounded-xl bg-slate-900 hover:bg-slate-800 text-white font-medium gap-2 h-10 px-5 transition-all">
            <Plus className="w-4 h-4" /> Add Source
          </Button>
        </div>
      </div>
    </div>
  );
}
