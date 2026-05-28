"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { Activity, ArrowRight, AlertCircle, Zap, Eye, EyeOff } from "lucide-react";
import { Input } from "@/components/ui/input";
import { authApi } from "@/lib/api-client";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [hasMounted, setHasMounted] = useState(false);

  useEffect(() => {
    setHasMounted(true);
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError("");

    try {
      const res = await authApi.login({ email, password });
      const { token, user } = res.data.data;
      localStorage.setItem("dmtool_token", token);
      localStorage.setItem("dmtool_user", JSON.stringify(user));
      router.push("/dashboard");

    } catch (err: any) {
      const errorData = err.response?.data?.error;
      const errorMessage = typeof errorData === 'object' ? errorData.message : errorData;
      setError(errorMessage || "Login failed. Please check your email and password.");
    } finally {
      setIsLoading(false);
    }
  };

  if (!hasMounted) return null;

  return (
    <div className="h-screen w-full flex overflow-hidden bg-white font-sans selection:bg-brand-500/10">
      
      {/* Left Panel: Simple Statement */}
      <div className="hidden lg:flex flex-1 flex-col justify-between p-16 bg-slate-900 relative overflow-hidden">
        <div className="relative z-10">
           <Link href="/" className="flex items-center gap-2 text-white font-bold text-lg tracking-tight">
             <div className="bg-white rounded p-1 text-slate-900">
               <Activity className="w-4 h-4" />
             </div>
             <span>DMTool</span>
           </Link>
        </div>

        <div className="relative z-10 max-w-sm">
           <h1 className="text-5xl font-bold tracking-tight text-white mb-6">
             Welcome back.
           </h1>
           <p className="text-lg text-slate-400 font-medium leading-relaxed">
             Sign in to manage your growth and see your latest dashboard updates.
           </p>
        </div>

        <div className="relative z-10">
           <p className="text-[10px] font-bold text-slate-700 uppercase tracking-widest">DMTool v4.0.2 / 2026</p>
        </div>

        {/* Subtle Background Dots */}
        <div className="absolute inset-0 opacity-10 pointer-events-none" 
             style={{ backgroundImage: 'radial-gradient(circle at 1px 1px, #ffffff 1px, transparent 0)', backgroundSize: '30px 30px' }} 
        />
      </div>


      {/* Right Panel: Form */}
      <div className="flex-1 flex flex-col justify-center items-center p-8 sm:p-12 relative">
        <div className="w-full max-w-sm space-y-8">
          
          <div className="lg:hidden mb-8">
             <Link href="/" className="flex items-center gap-2 text-slate-900 font-bold text-lg tracking-tight">
               <div className="bg-slate-900 rounded p-1 text-white">
                 <Activity className="w-4 h-4" />
               </div>
               <span>DMTool</span>
             </Link>
          </div>

          <div className="space-y-1">
             <h2 className="text-2xl font-bold text-slate-900">Sign In</h2>
             <p className="text-slate-500 text-sm font-medium">Enter your details to access your account.</p>
          </div>

          {error && (
            <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} className="p-3 rounded-xl bg-rose-50 text-rose-600 text-xs font-bold flex items-center gap-2 border border-rose-100 shadow-sm">
               <AlertCircle className="w-4 h-4 shrink-0" />
               {error}
            </motion.div>
          )}

          <form onSubmit={handleLogin} className="space-y-5">
            <div className="space-y-1.5">
               <label className="text-[10px] font-bold text-slate-400 uppercase tracking-widest ml-1">Email Address</label>
               <Input 
                  type="email" 
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="name@company.com"
                  className="h-12 rounded-xl bg-slate-50 border-slate-100 focus:bg-white focus:ring-slate-900/5 transition-all text-sm px-4"
               />
            </div>
            <div className="space-y-1.5">
               <div className="flex justify-between items-center px-1">
                  <label className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Password</label>
                  <Link href="#" className="text-[10px] font-bold text-slate-400 hover:text-slate-900 uppercase tracking-widest transition-colors">Forgot?</Link>
               </div>
               <div className="relative">
                  <Input 
                     type={showPassword ? "text" : "password"}
                     required
                     value={password}
                     onChange={(e) => setPassword(e.target.value)}
                     placeholder="Your password"
                     className="h-12 rounded-xl bg-slate-50 border-slate-100 focus:bg-white focus:ring-slate-900/5 transition-all text-sm px-4 pr-12"
                  />
                  <button 
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 transition-colors"
                  >
                    {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
               </div>
            </div>
            
            <button 
               type="submit" 
               disabled={isLoading}
               className="w-full h-12 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-xl shadow-lg shadow-slate-900/10 transition-all hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50 flex items-center justify-center gap-2"
            >
               {isLoading ? (
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
               ) : (
                  <>Sign In <ArrowRight className="w-4 h-4" /></>
               )}
            </button>
          </form>

          <p className="text-center text-xs font-bold text-slate-400">
             Don't have an account? <Link href="/register" className="text-slate-900 hover:underline">Create one here</Link>
          </p>
        </div>

        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-4 text-[10px] font-bold text-slate-300 uppercase tracking-widest whitespace-nowrap">
           <span>Secure</span>
           <span className="w-1 h-1 rounded-full bg-slate-200" />
           <span>Growth Focused</span>
        </div>
      </div>
    </div>
  );
}
