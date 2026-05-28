"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2, CheckCircle2, AlertCircle } from "lucide-react";
import { toast } from "@/components/ui/toaster";
import apiClient from "@/lib/api-client";

export default function IntegrationCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    const code = searchParams.get("code");
    const provider = searchParams.get("state");

    if (!code || !provider) {
      setStatus("error");
      setErrorMsg("Missing required parameters from OAuth provider.");
      return;
    }

    const completeIntegration = async () => {
      try {
        const endpoint = `/integrations/${provider}/callback?code=${code}`;
        await apiClient.get(endpoint);
        setStatus("success");
        toast(`${provider === 'meta' ? 'Meta' : 'Google'} integration successful!`, "success");
        setTimeout(() => router.push("/integrations"), 2000);
      } catch (err: any) {
        console.error("Integration failed", err);
        setStatus("error");
        setErrorMsg(err.response?.data?.error?.message || "Failed to complete integration.");
        toast("Integration failed", "error");
      }
    };

    completeIntegration();
  }, [searchParams, router]);

  return (
    <div className="flex flex-col items-center justify-center h-[60vh] text-center space-y-6">
      {status === "loading" && (
        <>
          <Loader2 className="w-12 h-12 animate-spin text-brand-600" />
          <div>
            <h2 className="text-2xl font-bold">Completing Integration</h2>
            <p className="text-slate-500 mt-2">Please wait while we securely save your credentials...</p>
          </div>
        </>
      )}

      {status === "success" && (
        <>
          <div className="w-20 h-20 rounded-full bg-emerald-100 flex items-center justify-center">
            <CheckCircle2 className="w-10 h-10 text-emerald-600" />
          </div>
          <div>
            <h2 className="text-2xl font-bold text-emerald-900">Connection Successful!</h2>
            <p className="text-slate-500 mt-2">Your account has been connected. Redirecting you back...</p>
          </div>
        </>
      )}

      {status === "error" && (
        <>
          <div className="w-20 h-20 rounded-full bg-rose-100 flex items-center justify-center">
            <AlertCircle className="w-10 h-10 text-rose-600" />
          </div>
          <div>
            <h2 className="text-2xl font-bold text-rose-900">Integration Failed</h2>
            <p className="text-rose-500 mt-2 font-medium">{errorMsg}</p>
            <button 
              onClick={() => router.push("/integrations")}
              className="mt-8 px-6 py-2 rounded-xl bg-slate-900 text-white font-bold hover:bg-slate-800 transition-colors"
            >
              Back to Integrations
            </button>
          </div>
        </>
      )}
    </div>
  );
}
