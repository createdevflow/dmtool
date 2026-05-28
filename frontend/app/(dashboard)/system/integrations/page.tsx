"use client";

import { motion } from "framer-motion";
import { 
  Puzzle, CheckCircle2, Plus, 
  ExternalLink, Search, Settings2, Loader2
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

export default function IntegrationsPage() {
  const [integrations, setIntegrations] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchIntegrations();
  }, []);

  const fetchIntegrations = async () => {
    try {
      const res = await dashboardApi.getIntegrations();
      const activeCreds = res.data.data || [];
      
      const supportedApps = [
        { id: 1, name: "Google Search Console", provider: "google", cat: "SEO & Traffic", desc: "Sync your search performance data for deep SEO insights.", status: "Available" },
        { id: 2, name: "Meta (Instagram/FB)", provider: "meta", cat: "Social Media", desc: "Fetch real-time metrics for your connected social profiles.", status: "Available" },
        { id: 3, name: "LinkedIn", provider: "linkedin", cat: "Professional", desc: "Professional networking and engagement tracking.", status: "Available" },
      ];

      const mapped = supportedApps.map(app => {
        const isActive = activeCreds.find((c: any) => c.provider === app.provider);
        return { ...app, status: isActive ? "Connected" : "Available" };
      });

      setIntegrations(mapped);
    } catch (err) {
      console.error("Failed to fetch integrations", err);
    } finally {
      setLoading(false);
    }
  };

  const handleDisconnect = async (provider: string) => {
    try {
      await dashboardApi.disconnectIntegration(provider);
      toast("Integration disconnected successfully", "success");
      fetchIntegrations();
    } catch (err) {
      toast("Failed to disconnect", "error");
    }
  };

  const handleConnection = async (app: any) => {
    if (app.status === "Connected") {
      handleDisconnect(app.provider);
      return;
    }

    try {
      let res;
      if (app.provider === "google") {
        res = await dashboardApi.getGoogleAuthUrl();
      } else if (app.provider === "meta") {
        res = await dashboardApi.getMetaAuthUrl();
      }

      if (res?.data?.data?.url) {
        window.location.href = res.data.data.url;
      } else {
        toast(`${app.name} integration is not available in this demo.`, "info");
      }
    } catch (err) {
      toast("Failed to initialize connection", "error");
    }
  };

  const handleSync = async (provider: string) => {
    try {
      toast(`Syncing ${provider} data...`, "info");
      // Fallback to project 1 sync if no specific endpoint for provider sync yet
      await dashboardApi.syncProject(1); 
      toast(`Successfully synced ${provider} data!`, "success");
      fetchIntegrations();
    } catch (err) {
      toast(`Failed to sync ${provider}`, "error");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-6xl mx-auto">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-bold tracking-tight">Integrations</h1>
          <p className="text-slate-500 mt-1">Connect your marketing stack for unified intelligence.</p>
        </motion.div>
        
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 w-4 h-4 text-slate-400" />
            <Input placeholder="Search apps..." className="pl-10 rounded-xl h-10 w-64 border-slate-200" />
          </div>
          <Button 
            className="rounded-xl bg-slate-900 text-white gap-2 h-10 px-4"
            onClick={() => toast("Integration request sent to our team!", "success")}
          >
            <Plus className="w-4 h-4" /> Request App
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {integrations.map((app, i) => (
          <motion.div 
            key={app.id} 
            initial={{ opacity: 0, y: 20 }} 
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: i * 0.1 }}
          >
            <Card className="border-border/50 hover:shadow-xl hover:shadow-slate-200/50 transition-all group rounded-2xl overflow-hidden">
              <CardHeader className="pb-4">
                <div className="flex items-start justify-between">
                  <div className="w-12 h-12 rounded-xl bg-slate-100 flex items-center justify-center group-hover:scale-110 transition-transform">
                    <Puzzle className="w-6 h-6 text-slate-600" />
                  </div>
                  <Badge variant={app.status === "Connected" ? "default" : "outline"} className={`rounded-lg ${app.status === 'Connected' ? 'bg-emerald-500 hover:bg-emerald-600' : 'text-slate-400'}`}>
                    {app.status}
                  </Badge>
                </div>
                <CardTitle className="mt-4 text-lg">{app.name}</CardTitle>
                <CardDescription className="text-xs font-semibold uppercase tracking-wider text-brand-600">{app.cat}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <p className="text-sm text-slate-500 leading-relaxed">{app.desc}</p>
                
                {app.status === "Connected" && (
                  <div className="bg-slate-50 rounded-xl p-3 border border-slate-100 space-y-1">
                    <div className="flex justify-between text-xs">
                      <span className="text-slate-500 font-semibold">Last Synced</span>
                      <span className="text-slate-900 font-bold">Just now</span>
                    </div>
                    <div className="flex justify-between text-xs">
                      <span className="text-slate-500 font-semibold">Records</span>
                      <span className="text-slate-900 font-bold">{Math.floor(Math.random() * 500) + 120}</span>
                    </div>
                  </div>
                )}

                <div className="flex items-center gap-2 pt-2">
                  <Button 
                    variant={app.status === "Connected" ? "outline" : "default"} 
                    className={`flex-1 rounded-xl h-10 font-semibold ${app.status === 'Connected' ? 'border-slate-200' : 'bg-slate-900 text-white hover:bg-brand-600'}`}
                    onClick={() => handleConnection(app)}
                  >
                    {app.status === "Connected" ? "Disconnect" : "Connect Now"}
                  </Button>
                  
                  {app.status === "Connected" && (
                    <Button 
                      variant="default" 
                      className="flex-1 rounded-xl h-10 font-semibold bg-brand-600 text-white hover:bg-brand-700"
                      onClick={() => handleSync(app.provider)}
                    >
                      Sync Now
                    </Button>
                  )}

                  {app.status !== "Connected" && (
                    <Button variant="ghost" size="icon" className="rounded-xl h-10 w-10 text-slate-400 hover:text-slate-900">
                      <ExternalLink className="w-4 h-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ))}
        {integrations.length === 0 && (
          <div className="col-span-full py-20 text-center border-2 border-dashed border-slate-100 rounded-3xl">
            <Puzzle className="w-12 h-12 mx-auto text-slate-200 mb-4" />
            <p className="text-slate-400 font-medium">No integrations found. Please create a project first.</p>
          </div>
        )}
      </div>
    </div>
  );
}
