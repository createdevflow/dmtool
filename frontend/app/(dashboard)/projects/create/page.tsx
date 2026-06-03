"use client";

import { motion, AnimatePresence } from "framer-motion";
import { 
  Globe, ArrowLeft, CheckCircle2, Loader2, 
  Search, Users, Activity, Instagram, Twitter, Linkedin, Facebook,
  Plus, Sparkles, Target
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { dashboardApi } from "@/lib/api-client";

type Step = "goal" | "details";

export default function CreateProjectPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("goal");
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [goal, setGoal] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [authWaiting, setAuthWaiting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [activeSocials, setActiveSocials] = useState<string[]>([]);
  const [handles, setHandles] = useState({
    instagram: "",
    twitter: "",
    linkedin: "",
    facebook: "",
  });

  const toggleSocial = (platform: string) => {
    setActiveSocials(prev => 
      prev.includes(platform) ? [] : [platform]
    );
  };

  const handleGoalSelect = (g: string) => {
    setGoal(g);
    setStep("details");
  };

  const executeProjectCreation = async (formattedUrl: string) => {
    try {
      setLoading(true);
      await dashboardApi.onboard({
        name: name,
        goal: goal || "seo",
        url: formattedUrl,
        ig_handle: handles.instagram,
        twitter_handle: handles.twitter,
        fb_handle: activeSocials.includes('facebook') ? "auto" : "",
        linkedin_handle: handles.linkedin
      });
      window.dispatchEvent(new Event("dmtool_projects_updated"));
      setSuccess(true);
      setTimeout(() => router.push("/dashboard"), 1500);
    } catch (err) {
      console.error("Failed to create project", err);
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    let formattedUrl = url;
    if (formattedUrl && !formattedUrl.startsWith("http")) {
      formattedUrl = `https://${formattedUrl}`;
    }

    const needsSocialLogin = (goal === 'social' || goal === 'both') && (activeSocials.includes('facebook') || activeSocials.includes('instagram') || activeSocials.includes('linkedin'));
    
    if (!needsSocialLogin) {
      executeProjectCreation(formattedUrl);
      return;
    }

    // Handle OAuth Flow
    let loginWindow = window.open('', 'SocialLogin', 'width=600,height=700,left=200,top=100');
    if (loginWindow) {
      loginWindow.document.write('<div style="font-family:sans-serif;padding:20px;text-align:center;">Connecting to Social Provider...</div>');
    }

    setAuthWaiting(true);

    try {
      let authUrl = null;
      if (activeSocials.includes("linkedin")) {
         const res = await dashboardApi.getLinkedinAuthUrl();
         authUrl = res?.data?.data?.url;
      } else {
         const res = await dashboardApi.getMetaAuthUrl();
         authUrl = res?.data?.data?.url;
      }

      if (authUrl && loginWindow) {
        loginWindow.location.href = authUrl;
        
        // Listen for success message
        const messageHandler = (event: MessageEvent) => {
          if (event.data?.type === 'OAUTH_SUCCESS') {
            window.removeEventListener('message', messageHandler);
            clearInterval(checkWindowTimer);
            setAuthWaiting(false);
            executeProjectCreation(formattedUrl);
          }
        };
        window.addEventListener('message', messageHandler);

        // Check if user manually closed popup
        const checkWindowTimer = setInterval(() => {
          if (loginWindow?.closed) {
            clearInterval(checkWindowTimer);
            window.removeEventListener('message', messageHandler);
            if (authWaiting) { // If it was still waiting
               setAuthWaiting(false);
               setLoading(false);
            }
          }
        }, 500);
      } else if (loginWindow) {
        loginWindow.close();
        setAuthWaiting(false);
        setLoading(false);
      }
    } catch (err) {
      console.error("Failed to get auth URL", err);
      if (loginWindow) loginWindow.close();
      setAuthWaiting(false);
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-12 pb-20">
      
      {/* 🧭 HEADER */}
      <div className="flex items-center justify-between">
        <motion.div initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }}>
          <Button variant="ghost" className="text-slate-500 hover:text-slate-900 -ml-2" onClick={() => step === "details" ? setStep("goal") : router.back()}>
            <ArrowLeft className="w-4 h-4 mr-2" /> 
            {step === "details" ? "Back to Goal" : "Back"}
          </Button>
        </motion.div>

        <div className="flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-100 border border-slate-200 shadow-sm">
           <div className={`w-1.5 h-1.5 rounded-full ${step === 'goal' ? 'bg-slate-900' : 'bg-emerald-500'}`} />
           <div className={`w-1.5 h-1.5 rounded-full ${step === 'details' ? 'bg-slate-900' : 'bg-slate-200'}`} />
           <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest ml-1">
             Step {step === 'goal' ? '1' : '2'} of 2
           </span>
        </div>
      </div>

      <AnimatePresence mode="wait">
        {step === "goal" ? (
          <motion.div 
            key="goal-step"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="space-y-12"
          >
            <div className="text-center space-y-4">
               <h1 className="text-4xl font-bold tracking-tight text-slate-900">Choose your focus.</h1>
               <p className="text-slate-500 text-lg font-medium max-w-md mx-auto">What is the primary objective for this new project?</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {[
                { id: "seo", title: "Website & SEO", icon: Search, desc: "Track traffic, rankings, and technical health", color: "text-blue-600 bg-blue-50" },
                { id: "social", title: "Social Media Intelligence", icon: Users, desc: "Analyze engagement, audience, and trends", color: "text-purple-600 bg-purple-50" },
                { id: "both", title: "Comprehensive Tracking", icon: Sparkles, desc: "Unified analytics for website and social", color: "text-amber-600 bg-amber-50" }
              ].map((item) => (
                <button 
                  key={item.id}
                  onClick={() => handleGoalSelect(item.id)}
                  className="group relative p-8 rounded-3xl bg-white border border-slate-200 hover:border-slate-900 hover:shadow-2xl hover:shadow-slate-200/50 transition-all duration-300 text-left"
                >
                  <div className={`w-12 h-12 rounded-2xl flex items-center justify-center mb-6 group-hover:bg-slate-900 group-hover:text-white transition-all duration-300 ${item.color}`}>
                    <item.icon className="w-6 h-6" />
                  </div>
                  <h3 className="text-xl font-bold text-slate-900 mb-2">{item.title}</h3>
                  <p className="text-sm text-slate-500 font-medium">{item.desc}</p>
                </button>
              ))}
            </div>
          </motion.div>
        ) : (
          <motion.div 
            key="details-step"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="space-y-10"
          >
            <div className="text-center space-y-4">
               <h1 className="text-4xl font-bold tracking-tight text-slate-900">Asset details.</h1>
               <p className="text-slate-500 text-lg font-medium">Connect the domain and social handles.</p>
            </div>

            <Card className="max-w-2xl mx-auto border-slate-200 shadow-2xl shadow-slate-200/40 rounded-[32px] overflow-hidden">
              <CardContent className="p-10">
                <form onSubmit={handleSubmit} className="space-y-8">
                  
                  <div className="space-y-6">
                    <div className="space-y-3">
                      <Label htmlFor="name" className="text-[11px] font-bold text-slate-400 uppercase tracking-widest ml-1">Project Name</Label>
                      <Input 
                        id="name" 
                        placeholder="My Awesome SaaS" 
                        required 
                        className="h-12 rounded-xl bg-slate-50 border-transparent focus:bg-white focus:border-slate-200 transition-all font-bold text-base px-5"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                      />
                    </div>

                    <div className="space-y-3">
                      <Label htmlFor="url" className="text-[11px] font-bold text-slate-400 uppercase tracking-widest ml-1">
                        Website URL {goal === 'social' ? '(Optional)' : ''}
                      </Label>
                      <div className="relative">
                        <Globe className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                        <Input 
                          id="url" 
                          placeholder={goal === 'social' ? "example.com (optional)" : "example.com"} 
                          required={goal !== 'social'}
                          className="h-12 rounded-xl bg-slate-50 border-transparent focus:bg-white focus:border-slate-200 transition-all font-bold text-base pl-12 pr-5"
                          value={url}
                          onChange={(e) => setUrl(e.target.value)}
                        />
                      </div>
                    </div>
                  </div>

                  {goal !== 'seo' && (
                    <div className="space-y-4 pt-4 border-t border-slate-100">
                      <Label className="text-[11px] font-bold text-slate-400 uppercase tracking-widest ml-1">Social Channels</Label>
                      <div className="grid grid-cols-4 gap-4">
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
                             className={`h-16 rounded-2xl border transition-all duration-300 flex flex-col items-center justify-center gap-1.5 ${
                               activeSocials.includes(s.id) 
                               ? 'bg-slate-900 border-slate-900 text-white' 
                               : 'bg-white border-slate-100 text-slate-400 hover:border-slate-300 shadow-sm'
                             }`}
                           >
                              <s.icon className="w-5 h-5" />
                              <span className="text-[9px] font-bold uppercase tracking-tight">{s.id}</span>
                           </button>
                         ))}
                      </div>

                      {activeSocials.length > 0 && (
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                          {activeSocials.includes("instagram") && (
                            <div className="space-y-2">
                              <Label>Instagram Handle</Label>
                              <Input
                                value={handles.instagram}
                                onChange={(e) => setHandles({...handles, instagram: e.target.value})}
                                placeholder="username"
                                className="rounded-xl border-slate-200 dark:border-slate-800"
                              />
                            </div>
                          )}
                          {activeSocials.includes("twitter") && (
                            <div className="space-y-2">
                              <Label>Twitter / X Handle</Label>
                              <Input
                                value={handles.twitter}
                                onChange={(e) => setHandles({...handles, twitter: e.target.value})}
                                placeholder="username"
                                className="rounded-xl border-slate-200 dark:border-slate-800"
                              />
                            </div>
                          )}
                          {activeSocials.includes("linkedin") && (
                            <div className="space-y-2">
                              <Label>LinkedIn Page URL</Label>
                              <Input
                                value={handles.linkedin}
                                onChange={(e) => setHandles({...handles, linkedin: e.target.value})}
                                placeholder="linkedin.com/company/..."
                                className="rounded-xl border-slate-200 dark:border-slate-800"
                              />
                            </div>
                          )}
                          {activeSocials.includes("facebook") && (
                            <div className="space-y-2">
                              <Label>Facebook Page URL</Label>
                              <Input
                                value={handles.facebook}
                                onChange={(e) => setHandles({...handles, facebook: e.target.value})}
                                placeholder="facebook.com/page"
                                className="rounded-xl border-slate-200 dark:border-slate-800"
                              />
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )}

                  <Button 
                    type="submit" 
                    className="w-full bg-slate-900 hover:bg-slate-800 text-white rounded-2xl h-14 font-bold text-lg transition-all disabled:opacity-70 shadow-xl shadow-slate-900/10"
                    disabled={loading || success || authWaiting}
                  >
                    {authWaiting ? (
                      <span className="flex items-center gap-2"><Loader2 className="w-5 h-5 animate-spin" /> Waiting for Authorization...</span>
                    ) : loading ? (
                      <Loader2 className="w-5 h-5 animate-spin" />
                    ) : success ? (
                      <span className="flex items-center gap-2"><CheckCircle2 className="w-5 h-5" /> Project Active!</span>
                    ) : (
                      "Launch Project"
                    )}
                  </Button>
                </form>
              </CardContent>
            </Card>
          </motion.div>
        )}
      </AnimatePresence>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 pt-8">
        {[
          { title: "Deep SEO Audit", desc: "Technical health & structure", icon: <Target className="w-4 h-4" /> },
          { title: "Social Sync", desc: "Live engagement monitoring", icon: <Activity className="w-4 h-4" /> },
          { title: "AI Strategy", desc: "Automated growth roadmap", icon: <Sparkles className="w-4 h-4" /> }
        ].map((feat, i) => (
          <motion.div 
            key={i}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 + (i * 0.1) }}
            className="p-5 rounded-2xl bg-white border border-slate-100 flex items-start gap-4"
          >
            <div className="w-8 h-8 rounded-lg bg-slate-50 flex items-center justify-center text-slate-400">
               {feat.icon}
            </div>
            <div>
              <h4 className="font-bold text-sm text-slate-900">{feat.title}</h4>
              <p className="text-[11px] text-slate-500 font-medium mt-0.5">{feat.desc}</p>
            </div>
          </motion.div>
        ))}
      </div>

    </div>
  );
}
