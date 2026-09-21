"use client";

import { motion } from "framer-motion";
import { MessageSquare, Send, User, Bot } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { dashboardApi } from "@/lib/api-client";

interface Message {
  role: "user" | "bot";
  text: string;
  time: string;
  source?: string;
}

function nowTime() {
  return new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export default function AIChatPage() {
  const [project, setProject] = useState<{ id: number; name?: string } | null>(null);
  const [ready, setReady] = useState(false);
  const [messages, setMessages] = useState<Message[]>([
    {
      role: "bot",
      text: "Drafts a caption from your topic. Not a strategy assistant. Type a topic and I’ll return one caption variant.",
      time: nowTime(),
    },
  ]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    (async () => {
      try {
        const pRes = await dashboardApi.getProjects();
        const projects = pRes.data?.data ?? pRes.data ?? [];
        if (Array.isArray(projects) && projects.length > 0) {
          let savedId = 0;
          try {
            savedId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0", 10);
          } catch {
            savedId = 0;
          }
          const selected =
            projects.find((p: { id: number }) => p.id === savedId) || projects[projects.length - 1];
          if (selected?.id) {
            localStorage.setItem("dmtool_active_project_id", String(selected.id));
            setProject(selected);
          }
        }
      } catch (err) {
        console.error(err);
      } finally {
        setReady(true);
      }
    })();
  }, []);

  const canSend = ready && !!project?.id && !!input.trim() && !loading;

  const handleSend = async () => {
    if (!input.trim() || loading) return;
    if (!project?.id) return;

    const userMsg: Message = { role: "user", text: input, time: nowTime() };
    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setLoading(true);

    try {
      const res = await dashboardApi.generateContent({
        project_id: project.id,
        topic: userMsg.text,
        platform: "blog",
        tone: "professional",
      });
      const data = res.data?.data ?? res.data;
      const variants = data?.variants ?? (Array.isArray(data) ? data : []);
      const source = typeof data?.source === "string" ? data.source : "";
      const responseText =
        variants.length > 0
          ? variants[0].content
          : "No caption was returned. Try the Content Generator for more variants.";

      setMessages((prev) => [
        ...prev,
        { role: "bot", text: responseText, time: nowTime(), source },
      ]);
    } catch (err) {
      console.error(err);
      setMessages((prev) => [
        ...prev,
        {
          role: "bot",
          text: "Caption generation failed. Check that you have a project selected.",
          time: nowTime(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-180px)] max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <motion.div initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            AI Chat <MessageSquare className="w-6 h-6 text-slate-400" />
          </h1>
          <p className="text-slate-500 mt-1">Drafts a caption. Not a strategy assistant.</p>
          {project?.name && (
            <p className="text-xs text-slate-400 mt-1">Using project: {project.name}</p>
          )}
        </motion.div>
      </div>

      <Card className="flex-1 border-border/50 shadow-xl shadow-slate-200/20 dark:shadow-none overflow-hidden flex flex-col rounded-3xl">
        <CardContent className="flex-1 overflow-y-auto p-6 space-y-6 subtle-scrollbar">
          {!ready ? null : !project ? (
            <div className="flex flex-col items-center justify-center h-full text-center text-slate-500 gap-3">
              <p>Create a project first. Chat needs a project id to generate a caption.</p>
              <Link href="/projects/create">
                <Button className="rounded-xl">Create project</Button>
              </Link>
            </div>
          ) : (
            <>
              {messages.map((msg, i) => (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
                >
                  <div className={`flex gap-4 max-w-[80%] ${msg.role === "user" ? "flex-row-reverse" : "flex-row"}`}>
                    <div
                      className={`w-10 h-10 rounded-2xl shrink-0 flex items-center justify-center ${
                        msg.role === "user"
                          ? "bg-brand-600 text-white"
                          : "bg-slate-100 dark:bg-slate-800 text-brand-600"
                      }`}
                    >
                      {msg.role === "user" ? <User className="w-5 h-5" /> : <Bot className="w-5 h-5" />}
                    </div>
                    <div className={`space-y-1 ${msg.role === "user" ? "items-end" : "items-start"}`}>
                      <div
                        className={`p-4 rounded-2xl text-sm leading-relaxed ${
                          msg.role === "user"
                            ? "bg-brand-600 text-white rounded-tr-none"
                            : "bg-slate-50 dark:bg-slate-900 border border-border/50 rounded-tl-none whitespace-pre-wrap"
                        }`}
                      >
                        {msg.text}
                      </div>
                      <p className="text-[10px] text-slate-400 font-bold uppercase tracking-widest px-1">
                        {msg.time}
                        {msg.source ? ` · ${msg.source === "ai" ? "AI" : "template"}` : ""}
                      </p>
                    </div>
                  </div>
                </motion.div>
              ))}
              {loading && (
                <div className="flex justify-start">
                  <div className="flex gap-4 max-w-[80%] flex-row">
                    <div className="w-10 h-10 rounded-2xl shrink-0 flex items-center justify-center bg-slate-100 dark:bg-slate-800 text-brand-600">
                      <Bot className="w-5 h-5 animate-pulse" />
                    </div>
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-900 border border-border/50 rounded-tl-none flex items-center gap-1.5 h-12">
                      <div className="w-1.5 h-1.5 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
                      <div className="w-1.5 h-1.5 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
                      <div className="w-1.5 h-1.5 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
                    </div>
                  </div>
                </div>
              )}
              <div ref={endRef} />
            </>
          )}
        </CardContent>

        <div className="p-4 bg-white dark:bg-slate-950 border-t border-border/50">
          <div className="relative max-w-4xl mx-auto">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && canSend && handleSend()}
              placeholder={project ? "Enter a caption topic…" : "Create a project first"}
              disabled={!project || loading}
              className="h-14 pl-6 pr-32 rounded-2xl border-slate-200 dark:border-slate-800 focus:ring-brand-500 shadow-lg shadow-brand-500/5 bg-slate-50/50 dark:bg-slate-900/50"
            />
            <div className="absolute right-2 top-2 flex gap-1">
              <Button
                disabled={!canSend}
                onClick={handleSend}
                className="h-10 px-6 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-bold gap-2 disabled:opacity-60"
              >
                Send <Send className="w-4 h-4" />
              </Button>
            </div>
          </div>
          <p className="text-[10px] text-center text-slate-400 mt-3 uppercase tracking-tighter font-bold">
            Uses the content generator. Source is labeled AI or template.
          </p>
        </div>
      </Card>
    </div>
  );
}
