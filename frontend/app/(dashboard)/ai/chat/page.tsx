"use client";

import { motion } from "framer-motion";
import { MessageSquare, Send, Sparkles, User, Bot, Zap, Plus, Search } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useState, useRef, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";

interface Message {
  role: "user" | "bot";
  text: string;
  time: string;
}

export default function AIChatPage() {
  const [messages, setMessages] = useState<Message[]>([
    { role: "bot", text: "Hello! I'm your DMTool AI Assistant. How can I help you optimize your marketing today?", time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }
  ]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || loading) return;
    
    const userMsg: Message = { role: "user", text: input, time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) };
    setMessages(prev => [...prev, userMsg]);
    setInput("");
    setLoading(true);

    try {
      const res = await dashboardApi.generateContent({
        project_id: 0, // Chat mode — no project required
        topic: userMsg.text,
        platform: "blog",
        tone: "professional",
      });
      const data = res.data?.data ?? res.data;
      const variants = data?.variants ?? (Array.isArray(data) ? data : []);
      const responseText = variants.length > 0 ? variants[0].content : "I can help with that — try the AI Content Generator for detailed results.";

      const botMsg: Message = {
        role: "bot",
        text: responseText,
        time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      };
      setMessages(prev => [...prev, botMsg]);
    } catch (err) {
      console.error(err);
      setMessages(prev => [...prev, { role: "bot", text: "I encountered an error trying to connect to the intelligence engine.", time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-180px)] max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <motion.div initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            AI Assistant <Sparkles className="w-6 h-6 text-brand-500 fill-brand-500/20" />
          </h1>
          <p className="text-slate-500 mt-1">Intelligent insights and automated marketing strategy.</p>
        </motion.div>
        <div className="flex gap-2">
           <Button variant="outline" className="rounded-xl border-slate-200 dark:border-slate-800 gap-2 h-10">
              <Plus className="w-4 h-4" /> New Chat
           </Button>
           <Button variant="outline" className="rounded-xl border-slate-200 dark:border-slate-800 gap-2 h-10">
              <Search className="w-4 h-4" /> History
           </Button>
        </div>
      </div>

      <Card className="flex-1 border-border/50 shadow-xl shadow-slate-200/20 dark:shadow-none overflow-hidden flex flex-col rounded-3xl">
        <CardContent className="flex-1 overflow-y-auto p-6 space-y-6 subtle-scrollbar">
           {messages.map((msg, i) => (
             <motion.div 
               key={i}
               initial={{ opacity: 0, y: 10 }}
               animate={{ opacity: 1, y: 0 }}
               transition={{ delay: i * 0.1 }}
               className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
             >
                <div className={`flex gap-4 max-w-[80%] ${msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'}`}>
                   <div className={`w-10 h-10 rounded-2xl shrink-0 flex items-center justify-center ${msg.role === 'user' ? 'bg-brand-600 text-white' : 'bg-slate-100 dark:bg-slate-800 text-brand-600'}`}>
                      {msg.role === 'user' ? <User className="w-5 h-5" /> : <Bot className="w-5 h-5" />}
                   </div>
                   <div className={`space-y-1 ${msg.role === 'user' ? 'items-end' : 'items-start'}`}>
                      <div className={`p-4 rounded-2xl text-sm leading-relaxed ${msg.role === 'user' ? 'bg-brand-600 text-white rounded-tr-none' : 'bg-slate-50 dark:bg-slate-900 border border-border/50 rounded-tl-none whitespace-pre-wrap'}`}>
                         {msg.text}
                      </div>
                      <p className="text-[10px] text-slate-400 font-bold uppercase tracking-widest px-1">{msg.time}</p>
                   </div>
                </div>
             </motion.div>
           ))}
           {loading && (
             <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="flex justify-start">
               <div className="flex gap-4 max-w-[80%] flex-row">
                  <div className="w-10 h-10 rounded-2xl shrink-0 flex items-center justify-center bg-slate-100 dark:bg-slate-800 text-brand-600">
                     <Bot className="w-5 h-5 animate-pulse" />
                  </div>
                  <div className="space-y-1 items-start">
                     <div className="p-4 rounded-2xl text-sm leading-relaxed bg-slate-50 dark:bg-slate-900 border border-border/50 rounded-tl-none flex items-center gap-1.5 h-12">
                        <div className="w-1.5 h-1.5 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                        <div className="w-1.5 h-1.5 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                        <div className="w-1.5 h-1.5 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                     </div>
                  </div>
               </div>
             </motion.div>
           )}
           <div ref={endRef} />
        </CardContent>

        <div className="p-4 bg-white dark:bg-slate-950 border-t border-border/50">
           <div className="relative max-w-4xl mx-auto">
              <Input 
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSend()}
                placeholder="Ask anything about your marketing data..." 
                className="h-14 pl-6 pr-32 rounded-2xl border-slate-200 dark:border-slate-800 focus:ring-brand-500 shadow-lg shadow-brand-500/5 bg-slate-50/50 dark:bg-slate-900/50"
              />
              <div className="absolute right-2 top-2 flex gap-1">
                 <Button variant="ghost" size="sm" className="h-10 w-10 p-0 rounded-xl text-slate-400 hover:text-brand-600">
                    <Zap className="w-5 h-5" />
                 </Button>
                 <Button disabled={!input.trim() || loading} onClick={handleSend} className="h-10 px-6 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-bold gap-2 disabled:opacity-60">
                    Send <Send className="w-4 h-4" />
                 </Button>
              </div>
           </div>
           <p className="text-[10px] text-center text-slate-400 mt-3 uppercase tracking-tighter font-bold">DMTool AI can make mistakes. Verify important financial metrics.</p>
        </div>
      </Card>
    </div>
  );
}
