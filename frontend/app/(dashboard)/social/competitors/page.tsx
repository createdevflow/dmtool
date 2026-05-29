"use client";

import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { Loader2, Zap, Users, ExternalLink, Image as ImageIcon, Search } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { motion } from "framer-motion";

interface RelatedProfile {
  handle: string;
  name: string;
  avatar: string;
  followers: number;
  following: number;
  posts: number;
  match_score: number;
  tags: string[];
  description: string;
}

export default function ProfileDiscoveryPage() {
  const [project, setProject] = useState<any>(null);
  const [mySocial, setMySocial] = useState<any>(null);
  const [relatedProfiles, setRelatedProfiles] = useState<RelatedProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [visibleCount, setVisibleCount] = useState(20);

  useEffect(() => {
    async function fetchData() {
      try {
        const pRes = await dashboardApi.getProjects();
        const allProjects = pRes.data.data || [];
        if (allProjects.length > 0) {
          const urlParams = new URLSearchParams(window.location.search);
          const targetProjectId = parseInt(urlParams.get("project_id") || "0");
          let savedProjectId = 0;
          try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch (e) {}
          const defaultProject = allProjects.find((p: any) => p.id === savedProjectId) || allProjects[allProjects.length - 1];
          const selected = targetProjectId
            ? allProjects.find((p: any) => p.id === targetProjectId) || defaultProject
            : defaultProject;
          if (selected) localStorage.setItem("dmtool_active_project_id", selected.id.toString());
          setProject(selected);

          // Fetch user's social insights
          try {
            const sRes = await dashboardApi.getSocialInsights(selected.id);
            const metrics = sRes.data.data || [];
            const active = metrics.find((m: any) => m.platform?.toLowerCase() === "instagram") || metrics[0];
            setMySocial(active);
          } catch (e) {
            console.error("Failed to load user social insights", e);
          }

          // Fetch related profiles
          try {
            const relRes = await dashboardApi.getRelatedProfiles(selected.id);
            if (relRes.data.data) {
              setRelatedProfiles(relRes.data.data);
            }
          } catch (e) {
            console.error("Failed to load related profiles", e);
          }
        }
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[70vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  if (!project || !project.ig_handle) {
    return (
      <div className="flex flex-col items-center justify-center h-[60vh] text-center max-w-md mx-auto">
        <div className="w-16 h-16 rounded-2xl bg-slate-50 flex items-center justify-center mb-6">
          <ImageIcon className="w-8 h-8 text-slate-300" />
        </div>
        <h2 className="text-xl font-bold text-slate-900 mb-2">Instagram Required</h2>
        <p className="text-slate-500 font-medium">Connect an Instagram account to your project to discover related profiles in your niche.</p>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto pb-12">
      {/* Header */}
      <div className="mb-8">
        <p className="text-xs font-bold tracking-widest text-slate-400 uppercase mb-2">Algorithm-Driven</p>
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold text-slate-900">Profile Discovery</h1>
        </div>
        <p className="text-slate-500 font-medium mt-3 text-sm max-w-2xl">
          We analyzed <span className="font-bold text-slate-700">@{project.ig_handle}</span>'s content, audience demographics, and posting style. Here are highly relevant profiles in your niche to draw inspiration from and analyze for competitive gaps.
        </p>
      </div>

      {relatedProfiles.length === 0 ? (
        <div className="rounded-3xl border-2 border-dashed border-slate-200 flex flex-col items-center justify-center py-24 text-center gap-4 bg-slate-50/50">
          <Search className="w-10 h-10 text-slate-300" />
          <div>
            <p className="text-slate-700 font-bold text-lg">No related profiles found</p>
            <p className="text-slate-500 text-sm mt-1 max-w-sm">We need more data from your connected Instagram account to build accurate algorithmic recommendations.</p>
          </div>
        </div>
      ) : (
        <>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {relatedProfiles.slice(0, visibleCount).map((profile, i) => (
            <motion.div
              key={profile.handle}
              initial={{ opacity: 0, y: 15 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: (i % 20) * 0.05 }}
            >
              <Card className="border border-slate-200 shadow-sm hover:shadow-md transition-shadow rounded-2xl overflow-hidden bg-white h-full group">
                <CardContent className="p-5">
                  {/* Header: Avatar + Stats side-by-side */}
                  <div className="flex items-center gap-5 mb-4">
                    {/* Left: Avatar with IG-like story ring */}
                    <div className="relative shrink-0">
                      <div className="absolute inset-0 rounded-full bg-gradient-to-tr from-yellow-400 via-rose-500 to-purple-500 p-[2px]">
                        <div className="h-full w-full rounded-full bg-white p-[2px]">
                          <Avatar className="w-[68px] h-[68px] rounded-full">
                            {profile.avatar ? (
                              <AvatarImage src={profile.avatar} alt={profile.handle} className="object-cover" />
                            ) : (
                              <AvatarFallback className="bg-slate-100 text-slate-500 font-bold uppercase">{profile.handle.slice(0, 2)}</AvatarFallback>
                            )}
                          </Avatar>
                        </div>
                      </div>
                      {/* Invisible spacer since absolute div doesn't take up layout space */}
                      <div className="w-[72px] h-[72px]"></div>
                    </div>
                    
                    {/* Right: Stats */}
                    <div className="flex flex-1 justify-around text-center">
                      <div className="flex flex-col items-center">
                        <p className="font-bold text-slate-900 text-[15px]">{profile.posts >= 1000 ? (profile.posts/1000).toFixed(1) + 'K' : profile.posts}</p>
                        <p className="text-[11px] font-medium text-slate-500 uppercase tracking-wider mt-0.5">Posts</p>
                      </div>
                      <div className="flex flex-col items-center">
                        <p className="font-bold text-slate-900 text-[15px]">
                          {profile.followers >= 1000000 ? (profile.followers / 1000000).toFixed(1) + 'M' : 
                           profile.followers >= 1000 ? (profile.followers / 1000).toFixed(1) + 'K' : 
                           profile.followers}
                        </p>
                        <p className="text-[11px] font-medium text-slate-500 uppercase tracking-wider mt-0.5">Followers</p>
                      </div>
                      <div className="flex flex-col items-center">
                        <p className="font-bold text-slate-900 text-[15px]">{profile.following >= 1000 ? (profile.following/1000).toFixed(1) + 'K' : profile.following}</p>
                        <p className="text-[11px] font-medium text-slate-500 uppercase tracking-wider mt-0.5">Following</p>
                      </div>
                    </div>
                  </div>

                  {/* Bio Section */}
                  <div className="mb-4 text-left">
                    <h3 className="font-bold text-slate-900 text-[15px] leading-tight">{profile.name}</h3>
                    <p className="text-[13px] font-medium text-slate-500 mb-2">@{profile.handle}</p>
                    <p className="text-[13px] text-slate-700 leading-snug">{profile.description}</p>
                    
                    {/* Tags rendered as hashtags */}
                    <div className="flex flex-wrap items-center gap-1.5 mt-2.5">
                      {profile.tags.map(tag => (
                        <span key={tag} className="text-[#00376b] font-medium text-[13px] hover:underline cursor-pointer">
                          #{tag.toLowerCase().replace(/\s+/g, '')}
                        </span>
                      ))}
                    </div>
                  </div>

                  {/* Buttons */}
                  <div className="flex items-center gap-2 mt-auto pt-2">
                    <a
                      href={`https://instagram.com/${profile.handle}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex-1 bg-slate-100 hover:bg-slate-200 text-slate-900 text-[13px] font-bold py-2 rounded-lg transition-colors text-center"
                    >
                      View Profile
                    </a>
                    <div className="flex items-center gap-1.5 bg-emerald-50 text-emerald-600 px-3 py-2 rounded-lg border border-emerald-100/50">
                      <Zap className="w-3.5 h-3.5 fill-emerald-600" />
                      <span className="text-[13px] font-bold">{profile.match_score}%</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
        
        {visibleCount < relatedProfiles.length && (
          <div className="mt-10 flex justify-center">
            <button 
              onClick={() => setVisibleCount(prev => prev + 20)}
              className="bg-slate-900 hover:bg-slate-800 text-white font-bold py-3 px-8 rounded-xl transition-colors shadow-sm"
            >
              See More Profiles
            </button>
          </div>
        )}
      </>
      )}
    </div>
  );
}
