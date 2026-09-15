"use client";

import { motion } from "framer-motion";
import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";

interface SimulatedBannerProps {
  message?: string;
}

/**
 * Shows a dismissible amber warning when data is seeded/estimated.
 * Drop this into any page that receives `is_simulated: true` from the API.
 */
export function SimulatedBanner({ message }: SimulatedBannerProps) {
  const router = useRouter();

  return (
    <motion.div
      initial={{ opacity: 0, height: 0 }}
      animate={{ opacity: 1, height: "auto" }}
      className="bg-amber-50 border border-amber-100 rounded-2xl p-4 flex items-center justify-between gap-4"
    >
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-amber-100 flex items-center justify-center text-amber-600 shrink-0">
          <AlertCircle className="w-4 h-4" />
        </div>
        <p className="text-sm text-amber-800 font-medium">
          <span className="font-bold">Estimated Data: </span>
          {message ?? "These metrics are estimated. Connect your accounts for live accuracy."}
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="rounded-lg bg-white border-amber-200 text-amber-700 hover:bg-amber-50 shrink-0"
        onClick={() => router.push("/integrations")}
      >
        Connect Accounts
      </Button>
    </motion.div>
  );
}
