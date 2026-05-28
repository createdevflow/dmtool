"use client";

import { motion, AnimatePresence } from "framer-motion";
import {
  Wand2, FileText, Copy, RefreshCw, Loader2, Sparkles,
  CheckCircle2, BarChart3, Info
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";

const PLATFORMS = [
  { id: "instagram",  label: "Instagram",  emoji: "📸" },
  { id: "linkedin",   label: "LinkedIn",   emoji: "💼" },
  { id: "twitter",    label: "Twitter/X",  emoji: "𝕏" },
  { id: "blog",       label: "Blog Post",  emoji: "📝" },
  { id: "facebook",   label: "Facebook",   emoji: "📘" },
  { id: "email",      label: "Email",      emoji: "📧" },
];

const TONES = ["Professional", "Casual", "Witty", "Persuasive", "Informative"];

export default function ContentGeneratorPage() {
  const [project, setProject] = useState<any>(null);
  const [topic, setTopic] = useState("");
  const [platform, setPlatform] = useState("instagram");
  const [tone, setTone] = useState("Professional");
  const [variants, setVariants] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState<number | null>(null);
  const [sourceNote, setSourceNote] = useState<string>("");

  useEffect(() => {
    (async () => {
      try {
        const pRes = await dashboardApi.getProjects();
        const projects = pRes.data?.data ?? pRes.data ?? [];
        if (projects.length > 0) setProject(projects[projects.length - 1]);
      } catch {}
    })();
  }, []);

  const handleGenerate = async () => {
    if (!topic.trim() || !project) return;
    setLoading(true);
    setVariants([]);
    setSourceNote("");
    try {
      const res = await dashboardApi.generateContent({
        project_id: project.id,
        topic: topic.trim(),
        platform,
        tone: tone.toLowerCase(),
      });
      const data = res.data?.data ?? res.data;
      const variantList = data?.variants ?? data ?? [];
      setVariants(Array.isArray(variantList) ? variantList : [variantList]);
      if (data?.source_note) setSourceNote(data.source_note);
    } catch (err) {
      console.error(err);
      setVariants([]);
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = (content: string, index: number) => {
    navigator.clipboard.writeText(content);
    setCopied(index);
    setTimeout(() => setCopied(null), 2000);
  };

  const ScoreBar = ({ label, value }: { label: string; value: number }) => (
    <div className="space-y-1">
      <div className="flex justify-between text-[10px] font-bold text-slate-400">
        <span>{label}</span>
        <span>{value}/10</span>
      </div>
      <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
        <motion.div initial={{ width: 0 }} animate={{ width: `${value * 10}%` }} transition={{ duration: 0.6, ease: "easeOut" }}
          className={`h-full rounded-full ${value >= 7 ? "bg-emerald-400" : value >= 5 ? "bg-amber-400" : "bg-rose-400"}`} />
      </div>
    </div>
  );

  return (
    <div className="space-y-8 max-w-6xl mx-auto pb-32">
      <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }}>
        <h1 className="text-3xl font-bold tracking-tight text-slate-900">AI Content Engine</h1>
        <p className="text-slate-500 mt-1">
          Generate high-converting copy for any platform. 3 unique variants every time.
        </p>
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-8 items-start">
        {/* Settings panel */}
        <div className="lg:col-span-2 space-y-5">
          <Card className="border-slate-100 shadow-sm rounded-3xl overflow-hidden">
            <CardHeader className="bg-slate-50/50 border-b border-slate-100 pb-4">
              <CardTitle className="text-base font-semibold text-slate-900">Settings</CardTitle>
            </CardHeader>
            <CardContent className="p-6 space-y-6">
              {/* Platform selector */}
              <div className="space-y-2">
                <Label className="text-sm font-semibold text-slate-700">Platform</Label>
                <div className="grid grid-cols-2 gap-2">
                  {PLATFORMS.map((p) => (
                    <button key={p.id} onClick={() => setPlatform(p.id)}
                      className={`flex items-center gap-2 px-3 py-2.5 rounded-xl border text-sm font-medium transition-all ${
                        platform === p.id
                          ? "border-slate-900 bg-slate-900 text-white"
                          : "border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50"
                      }`}>
                      <span>{p.emoji}</span> {p.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Topic input */}
              <div className="space-y-2">
                <Label htmlFor="topic" className="text-sm font-semibold text-slate-700">Topic</Label>
                <Textarea
                  id="topic"
                  placeholder="e.g. Why SEO is the best long-term investment for your business..."
                  className="min-h-[100px] rounded-2xl resize-none border-slate-200 focus:ring-slate-900 text-sm"
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                />
                <p className="text-[11px] text-slate-400">{topic.length} chars</p>
              </div>

              {/* Tone selector */}
              <div className="space-y-2">
                <Label className="text-sm font-semibold text-slate-700">Tone of Voice</Label>
                <div className="flex flex-wrap gap-2">
                  {TONES.map((t) => (
                    <button key={t} onClick={() => setTone(t)}
                      className={`px-3 py-1.5 rounded-full border text-xs font-semibold transition-all ${
                        tone === t
                          ? "border-slate-900 bg-slate-900 text-white"
                          : "border-slate-200 text-slate-500 hover:border-slate-400"
                      }`}>
                      {t}
                    </button>
                  ))}
                </div>
              </div>

              <Button
                onClick={handleGenerate}
                disabled={loading || !topic.trim() || !project}
                className="w-full h-12 bg-slate-900 hover:bg-slate-800 text-white rounded-2xl font-bold gap-2 disabled:opacity-60"
              >
                {loading
                  ? <><Loader2 className="w-5 h-5 animate-spin" /> Generating...</>
                  : <><Wand2 className="w-5 h-5" /> Generate 3 Variants</>
                }
              </Button>

              {!project && (
                <p className="text-xs text-amber-600 bg-amber-50 rounded-xl p-3">
                  Create a project first to use the content generator.
                </p>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Output panel */}
        <div className="lg:col-span-3 space-y-5">
          {sourceNote && (
            <div className="flex items-start gap-2 bg-blue-50 border border-blue-100 rounded-xl p-3">
              <Info className="w-4 h-4 text-blue-500 shrink-0 mt-0.5" />
              <p className="text-xs text-blue-700">{sourceNote}</p>
            </div>
          )}

          <AnimatePresence mode="wait">
            {loading && (
              <motion.div key="loading" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                className="bg-white border border-slate-100 rounded-3xl flex flex-col items-center justify-center py-24 gap-4">
                <div className="relative">
                  <div className="absolute inset-0 blur-xl bg-slate-900/10 animate-pulse rounded-full" />
                  <Sparkles className="w-12 h-12 text-slate-700 relative z-10 animate-pulse" />
                </div>
                <p className="text-sm font-bold uppercase tracking-widest text-slate-400 animate-pulse">
                  Generating content...
                </p>
              </motion.div>
            )}

            {!loading && variants.length > 0 && (
              <motion.div key="results" initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-5">
                {variants.map((variant: any, i: number) => (
                  <motion.div key={i} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.1 }}
                    className="bg-white border border-slate-100 rounded-3xl overflow-hidden shadow-sm hover:shadow-md transition-shadow">
                    <div className="flex items-center justify-between px-6 py-4 border-b border-slate-50 bg-slate-50/40">
                      <div className="flex items-center gap-3">
                        <span className="text-[11px] font-bold text-slate-400 uppercase tracking-widest">
                          Variant {variant.id ?? i + 1}
                        </span>
                        <Badge variant="outline" className="text-[10px] bg-white text-slate-500 border-slate-100">
                          {variant.char_count ?? variant.content?.length ?? 0} chars
                        </Badge>
                      </div>
                      <Button variant="ghost" size="sm"
                        className="rounded-lg h-8 gap-2 text-slate-500 hover:text-slate-900 text-[11px] font-bold"
                        onClick={() => handleCopy(variant.content, i)}>
                        {copied === i ? (
                          <><CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" /> Copied!</>
                        ) : (
                          <><Copy className="w-3.5 h-3.5" /> Copy</>
                        )}
                      </Button>
                    </div>

                    <div className="px-6 py-5">
                      <p className="whitespace-pre-wrap leading-relaxed text-slate-700 font-medium text-sm">
                        {variant.content}
                      </p>
                    </div>

                    {(variant.platform_fit != null || variant.clarity != null) && (
                      <div className="px-6 pb-5 space-y-2 border-t border-slate-50 pt-4">
                        <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest flex items-center gap-1.5">
                          <BarChart3 className="w-3 h-3" /> Quality Scores
                        </p>
                        <div className="grid grid-cols-3 gap-4">
                          <ScoreBar label="Platform Fit" value={variant.platform_fit ?? 0} />
                          <ScoreBar label="Clarity" value={variant.clarity ?? 0} />
                          <ScoreBar label="CTA Strength" value={variant.cta_strength ?? 0} />
                        </div>
                      </div>
                    )}
                  </motion.div>
                ))}

                <Button variant="outline"
                  onClick={handleGenerate}
                  className="w-full h-11 rounded-2xl border-slate-200 text-slate-600 font-semibold gap-2 hover:bg-slate-50">
                  <RefreshCw className="w-4 h-4" /> Regenerate Variants
                </Button>
              </motion.div>
            )}

            {!loading && variants.length === 0 && (
              <motion.div key="empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }}
                className="bg-white border border-slate-100 rounded-3xl flex flex-col items-center justify-center py-24 text-center">
                <FileText className="w-16 h-16 text-slate-200 stroke-[1px] mb-4" />
                <p className="text-slate-700 font-semibold">Your content will appear here</p>
                <p className="text-slate-400 text-sm mt-1.5">
                  Select a platform, enter your topic, and click Generate.
                </p>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}
