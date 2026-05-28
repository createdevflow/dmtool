"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence, Variants } from "framer-motion";
import { Globe, Users, ArrowRight, Activity, Search, AlertTriangle, CheckCircle2, Instagram, Twitter, Linkedin, Facebook, Loader2 } from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";

type OnboardingStep = 1 | 2 | 3 | 4;

export default function OnboardingPage() {
  const [step, setStep] = useState<OnboardingStep>(1);
  const [goal, setGoal] = useState<string>("");
  const [url, setUrl] = useState("");
  const [handles, setHandles] = useState({
    instagram: "",
    twitter: "",
    linkedin: "",
    facebook: "",
  });
  const [activeSocials, setActiveSocials] = useState<string[]>([]);
  const [loadingText, setLoadingText] = useState("Analyzing structure...");
  const [error, setError] = useState("");
  const [hasMounted, setHasMounted] = useState(false);

  const router = useRouter();

  useEffect(() => {
    setHasMounted(true);
  }, []);

  const handleGoalSelect = (selectedGoal: string) => {
    setGoal(selectedGoal);
    setStep(2);
  };

  const toggleSocial = (platform: string) => {
    setActiveSocials(prev => 
      prev.includes(platform) ? prev.filter(p => p !== platform) : [...prev, platform]
    );
  };

  const [projectData, setProjectData] = useState<any>(null);

  const handleAnalyze = async (e: React.FormEvent) => {
    e.preventDefault();
    setStep(3);
    
    setTimeout(() => setLoadingText("Extracting metadata..."), 1500);
    setTimeout(() => setLoadingText("Mapping brand identity..."), 3000);
    setTimeout(() => setLoadingText("Building initial strategy..."), 4500);

    try {
      const res = await dashboardApi.onboard({
        url: url || `https://social-project-${Math.floor(Math.random() * 10000)}.com`,
        goal,
        ig_handle: handles.instagram,
        twitter_handle: handles.twitter,
        linkedin_handle: handles.linkedin,
        facebook_handle: handles.facebook
      });


      
      setProjectData(res.data.data.project);

      setTimeout(() => setStep(4), 6000);
    } catch (err: any) {
      console.error(err);
      const errorData = err.response?.data?.error;
      const errorMessage = typeof errorData === 'object' ? errorData.message : errorData;
      setError(errorMessage || "Analysis failed. Please check your inputs.");
      setStep(2);
    }
  };


  const containerVariants: Variants = {
    hidden: { opacity: 0, y: 20 },
    visible: { 
      opacity: 1, 
      y: 0,
      transition: { duration: 0.8, ease: [0.22, 1, 0.36, 1] }
    },
    exit: { 
      opacity: 0, 
      scale: 0.98,
      transition: { duration: 0.4 }
    }
  };

  if (!hasMounted) return null;

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-white overflow-hidden relative selection:bg-brand-500/10">
      
      {/* Subtle Background Elements */}
      <div className="absolute inset-0 pointer-events-none opacity-40">
        <div className="absolute top-0 left-0 w-full h-full bg-[radial-gradient(circle_at_20%_20%,_#f8fafc_0%,_transparent_50%)]" />
        <div className="absolute bottom-0 right-0 w-full h-full bg-[radial-gradient(circle_at_80%_80%,_#f1f5f9_0%,_transparent_50%)]" />
      </div>

      <div className="w-full max-w-4xl px-6 py-12 relative z-10">

        <AnimatePresence mode="wait">
          
          {/* STEP 1: Goal Selection */}
          {step === 1 && (
            <motion.div 
              key="step1" 
              variants={containerVariants}
              initial="hidden"
              animate="visible"
              exit="exit"
              className="space-y-16"
            >
              <div className="text-center space-y-4">
                <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-slate-900 text-white font-bold text-[10px] uppercase tracking-widest mb-4">
                   <Activity className="w-3 h-3" /> System Ready
                </div>
                <h1 className="text-5xl md:text-7xl font-bold text-slate-900 tracking-tight leading-[0.9]">
                  What is your <br />
                  <span className="text-slate-300">primary goal?</span>
                </h1>
                <p className="text-lg text-slate-500 font-medium max-w-md mx-auto">
                  Select the core focus for your digital acceleration.
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {[
                  { id: "seo", title: "Search", icon: Search, desc: "SEO Dominance", color: "bg-slate-50" },
                  { id: "social", title: "Social", icon: Users, desc: "Brand Growth", color: "bg-slate-50" },
                  { id: "both", title: "Combined", icon: Globe, desc: "Unified Strategy", color: "bg-slate-50" }
                ].map((item) => (
                  <button 
                    key={item.id}
                    onClick={() => handleGoalSelect(item.id)}
                    className="group relative p-10 rounded-3xl bg-white border border-slate-100 hover:border-slate-900 transition-all duration-500 text-left"
                  >
                    <div className="w-12 h-12 rounded-2xl bg-slate-50 flex items-center justify-center mb-6 group-hover:bg-slate-900 group-hover:text-white transition-colors duration-500">
                      <item.icon className="w-6 h-6" />
                    </div>
                    <h3 className="text-xl font-bold text-slate-900 mb-2">{item.title}</h3>
                    <p className="text-sm text-slate-500 font-medium">{item.desc}</p>
                    <div className="absolute bottom-10 right-10 opacity-0 group-hover:opacity-100 translate-x-2 group-hover:translate-x-0 transition-all">
                      <ArrowRight className="w-5 h-5 text-slate-900" />
                    </div>
                  </button>
                ))}
              </div>
            </motion.div>
          )}

          {/* STEP 2: Website & Social Input */}
          {step === 2 && (
            <motion.div 
              key="step2" 
              variants={containerVariants}
              initial="hidden"
              animate="visible"
              exit="exit"
              className="space-y-12"
            >
              <div className="text-center space-y-4">
                <h1 className="text-5xl font-bold text-slate-900 tracking-tight">Connect Assets.</h1>
                <p className="text-slate-500 text-lg font-medium">Where should we start the analysis?</p>
              </div>

              <div className="max-w-2xl mx-auto w-full">
                {error && (
                  <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }} className="mb-6 p-4 rounded-xl bg-rose-50 text-rose-600 text-sm font-bold flex items-center gap-3 border border-rose-100">
                    <AlertTriangle className="w-5 h-5 shrink-0" />
                    {error}
                  </motion.div>
                )}
                <form onSubmit={handleAnalyze} className="space-y-10">

                  {/* Conditional Website URL Field */}
                  {(goal === "seo" || goal === "both") && (
                    <div className="space-y-3">
                      <label className="text-[10px] font-bold text-slate-400 uppercase tracking-widest ml-1">Website URL</label>
                      <div className="relative">
                         <Globe className="absolute left-5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                         <Input 
                          id="onboarding-url"
                          required
                          type="url"
                          placeholder="https://yourbrand.com"
                          value={url}
                          onChange={(e) => setUrl(e.target.value)}
                          className="h-12 rounded-xl bg-slate-50 border-transparent focus:bg-white focus:border-slate-200 transition-all font-bold text-base pl-14"
                         />

                      </div>
                    </div>
                  )}


                  {(goal === "social" || goal === "both") && (
                    <div className="space-y-4">
                      <label className="text-[10px] font-bold text-slate-400 uppercase tracking-widest ml-1">Social Channels</label>
                      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                         {[
                           { id: 'instagram', icon: Instagram },
                           { id: 'twitter', icon: Twitter },
                           { id: 'linkedin', icon: Linkedin },
                           { id: 'facebook', icon: Facebook }
                         ].map((s) => (
                           <button
                             key={s.id}
                             type="button"
                             onClick={() => toggleSocial(s.id)}
                             className={`h-20 rounded-2xl border transition-all duration-300 flex flex-col items-center justify-center gap-2 ${
                               activeSocials.includes(s.id) 
                               ? 'bg-slate-900 border-slate-900 text-white' 
                               : 'bg-white border-slate-100 text-slate-400 hover:border-slate-300'
                             }`}
                           >
                              <s.icon className="w-5 h-5" />
                              <span className="text-[10px] font-bold uppercase tracking-tight">{s.id}</span>
                           </button>
                         ))}
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
                         {activeSocials.map((platform) => (
                            <Input 
                              key={platform}
                              id={`social-input-${platform}`}
                              placeholder={`@${platform} username`}
                              className="h-12 rounded-xl bg-slate-50 border-transparent focus:bg-white focus:border-slate-200"
                              value={(handles as any)[platform]}
                              onChange={(e) => setHandles({...handles, [platform]: e.target.value})}
                            />
                         ))}

                      </div>
                    </div>
                  )}

                  <button 
                    type="submit"
                    className="w-full h-16 bg-slate-900 text-white hover:bg-slate-800 font-bold text-lg rounded-2xl shadow-xl shadow-slate-900/10 transition-all hover:scale-[1.01] active:scale-[0.99]"
                  >
                    Start Analysis
                  </button>
                </form>
              </div>
            </motion.div>
          )}

          {/* STEP 3: Loading / Analysis */}
          {step === 3 && (
            <motion.div 
              key="step3" 
              variants={containerVariants}
              initial="hidden"
              animate="visible"
              exit="exit"
              className="flex flex-col items-center justify-center space-y-8"
            >
              <div className="relative">
                <div className="w-24 h-24 rounded-full border-4 border-slate-100 border-t-slate-900 animate-spin" />
                <div className="absolute inset-0 flex items-center justify-center">
                   <Activity className="w-6 h-6 text-slate-900 animate-pulse" />
                </div>
              </div>
              <div className="text-center space-y-2">
                 <h2 className="text-3xl font-bold text-slate-900">{loadingText}</h2>
                 <p className="text-slate-400 font-medium">Building your strategy engine...</p>
              </div>
            </motion.div>
          )}

          {/* STEP 4: Success / Strategy Review */}
          {step === 4 && (
            <motion.div 
              key="step4" 
              variants={containerVariants}
              initial="hidden"
              animate="visible"
              className="space-y-12"
            >
              <div className="text-center space-y-4">
                <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-emerald-50 text-emerald-600 font-bold text-xs uppercase tracking-widest mb-2">
                   <CheckCircle2 className="w-4 h-4" /> System Synchronized
                </div>
                <h1 className="text-5xl font-bold tracking-tight text-slate-900">Ready to Launch.</h1>
                <p className="text-slate-500 text-lg font-medium">
                  We've mapped out the first phase for <span className="text-slate-900 font-bold">{projectData?.name || 'your brand'}</span>.
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-8 items-stretch max-w-4xl mx-auto">
                <div className="bg-slate-50 rounded-3xl p-10 border border-slate-100">
                  <h3 className="font-bold text-slate-400 flex items-center gap-2 mb-8 text-[10px] uppercase tracking-[0.2em]">
                     <AlertTriangle className="w-4 h-4" /> Initial Focus
                  </h3>
                  <ul className="space-y-6">
                    {(goal === "seo" || goal === "both") && (
                      <>
                        <li className="flex items-start gap-4 text-slate-900 font-bold">
                          <span className="w-1.5 h-1.5 rounded-full bg-slate-900 shrink-0 mt-2" />
                          Optimize core site structure
                        </li>
                        <li className="flex items-start gap-4 text-slate-900 font-bold">
                          <span className="w-1.5 h-1.5 rounded-full bg-slate-900 shrink-0 mt-2" />
                          Fix technical indexing issues
                        </li>
                      </>
                    )}
                    {(goal === "social" || goal === "both") && (
                      <>
                        <li className="flex items-start gap-4 text-slate-900 font-bold">
                          <span className="w-1.5 h-1.5 rounded-full bg-slate-900 shrink-0 mt-2" />
                          Calibrate engagement frequency
                        </li>
                        <li className="flex items-start gap-4 text-slate-900 font-bold">
                          <span className="w-1.5 h-1.5 rounded-full bg-slate-900 shrink-0 mt-2" />
                          Align brand voice across channels
                        </li>
                      </>
                    )}
                  </ul>
                </div>

                <div className="bg-slate-900 rounded-3xl p-10 text-white flex flex-col justify-between shadow-2xl relative overflow-hidden group">
                   <div className="relative z-10">
                     <h3 className="text-2xl font-bold mb-4">Command Center</h3>
                     <p className="text-slate-400 font-medium leading-relaxed mb-10">
                        Your workspace is now active with live data feeds.
                     </p>
                     
                     <div className="bg-white/5 rounded-2xl p-6 border border-white/10 mb-10">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-2 block">First Action</span>
                        <p className="text-lg font-bold">Review AI Recommendations</p>
                     </div>
                   </div>

                   <button 
                    onClick={() => router.push("/dashboard")}
                    className="w-full h-16 bg-white text-slate-900 hover:bg-slate-50 font-bold text-lg rounded-2xl shadow-xl transition-all hover:scale-[1.01] active:scale-[0.99] relative z-10"
                   >
                     Enter Dashboard
                   </button>
                   
                   <div className="mt-4 flex gap-2 relative z-10">
                     <button
                       onClick={async () => {
                         try {
                           const res = await dashboardApi.getGoogleAuthUrl();
                           if (res?.data?.data?.url) window.location.href = res.data.data.url;
                         } catch (err) {
                           toast("Google connection currently unavailable", "error");
                         }
                       }}
                       className="flex-1 h-12 bg-white/10 hover:bg-white/20 text-white font-bold text-sm rounded-xl transition-all border border-white/20 flex items-center justify-center gap-2"
                     >
                        Connect Google
                     </button>
                     <button
                       onClick={async () => {
                         try {
                           const res = await dashboardApi.getMetaAuthUrl();
                           if (res?.data?.data?.url) window.location.href = res.data.data.url;
                         } catch (err) {
                           toast("Meta connection currently unavailable", "error");
                         }
                       }}
                       className="flex-1 h-12 bg-white/10 hover:bg-white/20 text-white font-bold text-sm rounded-xl transition-all border border-white/20 flex items-center justify-center gap-2"
                     >
                        Connect Meta
                     </button>
                   </div>
                </div>
              </div>
            </motion.div>
          )}

        </AnimatePresence>
      </div>
    </div>
  );
}
