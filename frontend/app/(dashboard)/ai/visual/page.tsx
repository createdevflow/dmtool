"use client";

import { motion } from "framer-motion";
import { ImageIcon, Wand2, Download, Layers, Palette, Maximize2, Loader2, Sparkles, Plus } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { useState } from "react";

const samples = [
  { id: 1, prompt: "Cyberpunk cityscape with neon marketing signs", img: "https://images.unsplash.com/photo-1614850523296-d8c1af93d400?q=80&w=300&auto=format&fit=crop" },
  { id: 2, prompt: "Minimalist SaaS dashboard 3D abstract", img: "https://images.unsplash.com/photo-1550745165-9bc0b252726f?q=80&w=300&auto=format&fit=crop" },
  { id: 3, prompt: "Holographic data visualization spheres", img: "https://images.unsplash.com/photo-1451187580459-43490279c0fa?q=80&w=300&auto=format&fit=crop" }
];

export default function VisualAIPage() {
  const [loading, setLoading] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [generatedImg, setGeneratedImg] = useState<string | null>(null);

  const handleGenerate = () => {
    setLoading(true);
    setGeneratedImg(null);
    // Simulate generation by choosing a high-quality relevant image
    setTimeout(() => {
      setGeneratedImg(`https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?q=80&w=800&auto=format&fit=crop`);
      setLoading(false);
    }, 2500);
  };

  return (
    <div className="space-y-8 max-w-6xl mx-auto">
      <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }}>
        <h1 className="text-3xl font-bold tracking-tight">Visual AI Studio</h1>
        <p className="text-slate-500 mt-1">Generate stunning marketing assets and brand visuals with GenAI.</p>
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <Card className="lg:col-span-1 border-border/50 h-fit rounded-3xl overflow-hidden shadow-sm">
           <CardHeader className="bg-slate-50/50 dark:bg-slate-900/50 border-b border-border/50 p-6">
              <CardTitle className="text-lg flex items-center gap-2">
                 Studio Controls <Palette className="w-4 h-4 text-brand-500" />
              </CardTitle>
           </CardHeader>
           <CardContent className="p-6 space-y-8">
              <div className="space-y-3">
                 <Label htmlFor="img-prompt">Image Prompt</Label>
                 <Input 
                   id="img-prompt" 
                   placeholder="Describe your vision..." 
                   className="rounded-xl h-12 border-slate-200 dark:border-slate-800"
                   value={prompt}
                   onChange={(e) => setPrompt(e.target.value)}
                 />
              </div>

              <div className="space-y-4">
                 <div className="flex justify-between items-center">
                    <Label>Creative Strength</Label>
                    <span className="text-[10px] font-bold text-brand-600 bg-brand-50 dark:bg-brand-900/20 px-2 py-0.5 rounded uppercase">Optimized</span>
                 </div>
                 <Slider defaultValue={[75]} max={100} step={1} className="py-2" />
              </div>

              <div className="space-y-3">
                 <Label>Aspect Ratio</Label>
                 <div className="grid grid-cols-3 gap-2">
                    {["1:1", "16:9", "4:5"].map((ratio) => (
                      <Button key={ratio} variant="outline" className={`rounded-xl h-10 text-xs font-bold ${ratio === '1:1' ? 'border-brand-500 bg-brand-50/50 text-brand-600' : ''}`}>
                         {ratio}
                      </Button>
                    ))}
                 </div>
              </div>

              <Button 
                className="w-full h-12 bg-slate-900 dark:bg-brand-600 hover:bg-brand-500 text-white rounded-2xl font-bold gap-2 shadow-lg shadow-brand-500/10"
                onClick={handleGenerate}
                disabled={loading || !prompt}
              >
                {loading ? <Loader2 className="w-5 h-5 animate-spin" /> : <><Sparkles className="w-5 h-5" /> Render Visual</>}
              </Button>
           </CardContent>
        </Card>

        <Card className="lg:col-span-2 border-border/50 bg-slate-50/30 dark:bg-slate-950/30 min-h-[600px] flex flex-col rounded-3xl overflow-hidden relative border-dashed border-2">
           {loading ? (
             <div className="flex-1 flex flex-col items-center justify-center p-12 space-y-6">
                <div className="relative w-24 h-24">
                   <div className="absolute inset-0 bg-brand-500/20 animate-ping rounded-full" />
                   <Wand2 className="w-24 h-24 text-brand-600 relative z-10" />
                </div>
                <div className="text-center">
                   <h3 className="text-lg font-bold">Synthesizing Pixels</h3>
                   <p className="text-sm text-slate-500 max-w-[280px] mt-1 italic">"The future of branding is built by algorithms, curated by humans."</p>
                </div>
             </div>
           ) : (
             <div className="flex-1 p-8">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 h-full">
                   <div className="md:col-span-2 relative group rounded-2xl overflow-hidden border border-border/50 aspect-video bg-white dark:bg-slate-900 shadow-xl">
                      {generatedImg ? (
                        <img src={generatedImg} alt="Generated" className="w-full h-full object-cover animate-in fade-in zoom-in duration-700" />
                      ) : (
                        <div className="absolute inset-0 flex flex-col items-center justify-center space-y-2 opacity-30">
                           <ImageIcon className="w-16 h-16" />
                           <p className="text-xs font-bold uppercase tracking-widest">Main Viewport</p>
                        </div>
                      )}
                      {generatedImg && (
                        <div className="absolute bottom-4 right-4 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                           <Button variant="secondary" size="sm" className="rounded-lg h-9 w-9 p-0"><Maximize2 className="w-4 h-4" /></Button>
                           <Button variant="secondary" size="sm" className="rounded-lg h-9 w-9 p-0"><Download className="w-4 h-4" /></Button>
                        </div>
                      )}
                   </div>
                   
                   <div className="space-y-4 md:col-span-2">
                      <h4 className="text-xs font-bold uppercase tracking-widest text-slate-400 mt-4">Inspiration Samples</h4>
                      <div className="grid grid-cols-3 gap-4">
                         {samples.map((s) => (
                           <div key={s.id} className="relative aspect-square rounded-xl overflow-hidden border border-border group cursor-pointer" onClick={() => setGeneratedImg(s.img)}>
                              <img src={s.img} alt={s.prompt} className="w-full h-full object-cover grayscale group-hover:grayscale-0 transition-all duration-500" />
                              <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                                 <Plus className="w-6 h-6 text-white" />
                              </div>
                           </div>
                         ))}
                      </div>
                   </div>
                </div>
             </div>
           )}
        </Card>
      </div>
    </div>
  );
}
