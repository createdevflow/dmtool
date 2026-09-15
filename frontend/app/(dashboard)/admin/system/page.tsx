"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Activity, ArrowLeft, CheckCircle, AlertTriangle, XCircle, RefreshCw
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { adminApi } from "@/lib/api-client";
import { cn } from "@/lib/utils";

type HealthData = {
  status: string;
  services: Record<string, {
    status: string;
    latency?: string;
    total?: number;
    expired?: number;
    sync_errors?: number;
    with_issues?: number;
  }>;
};

const SERVICE_LABELS: Record<string, string> = {
  api_server: "API Server",
  database: "Database",
  oauth_integrations: "OAuth Integrations",
  projects: "Projects",
};

const SERVICE_ICONS: Record<string, typeof Activity> = {
  api_server: Activity,
  database: Activity,
  oauth_integrations: Activity,
  projects: Activity,
};

export default function AdminSystemPage() {
  const [health, setHealth] = useState<HealthData | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchHealth = async (showRefresh = false) => {
    if (showRefresh) setRefreshing(true);
    try {
      const r = await adminApi.platformHealth();
      setHealth(r.data.data);
    } catch {
      // Non-critical.
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchHealth();
  }, []);

  const statusIcon = (status: string) => {
    switch (status) {
      case "healthy": return <CheckCircle className="w-5 h-5 text-emerald-500" />;
      case "degraded": return <AlertTriangle className="w-5 h-5 text-amber-500" />;
      case "down": return <XCircle className="w-5 h-5 text-red-500" />;
      default: return <Activity className="w-5 h-5 text-slate-400" />;
    }
  };

  const statusBadge = (status: string) => {
    switch (status) {
      case "healthy": return "bg-emerald-100 text-emerald-800";
      case "degraded": return "bg-amber-100 text-amber-800";
      case "down": return "bg-red-100 text-red-800";
      default: return "bg-slate-100 text-slate-700";
    }
  };

  if (loading) {
    return <div className="text-slate-400 py-12 text-center">Checking system health…</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Link
            href="/admin"
            className="text-xs text-slate-500 hover:text-slate-900 inline-flex items-center gap-1 mb-2"
          >
            <ArrowLeft className="w-3 h-3" /> Back to overview
          </Link>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Activity className="w-6 h-6 text-slate-700" />
            System Health
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            Service status and platform connectivity.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchHealth(true)}
          disabled={refreshing}
        >
          <RefreshCw className={cn("w-4 h-4 mr-2", refreshing && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {/* Overall status */}
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center gap-3">
            {statusIcon(health?.status || "unknown")}
            <div>
              <div className="text-lg font-semibold text-slate-900 capitalize">
                {health?.status || "Unknown"}
              </div>
              <div className="text-sm text-slate-500">Overall system status</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Service grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {health && Object.entries(health.services).map(([key, service]) => {
          const label = SERVICE_LABELS[key] || key;
          return (
            <Card key={key}>
              <CardContent className="p-5">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    {statusIcon(service.status)}
                    <div>
                      <div className="font-medium text-slate-900">{label}</div>
                      <Badge className={cn("text-[10px] mt-1", statusBadge(service.status))}>
                        {service.status}
                      </Badge>
                    </div>
                  </div>
                </div>
                <div className="mt-4 space-y-2 text-sm">
                  {service.latency && (
                    <div className="flex justify-between">
                      <span className="text-slate-500">Latency</span>
                      <span className="font-medium tabular-nums">{service.latency}</span>
                    </div>
                  )}
                  {service.total !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-slate-500">Total</span>
                      <span className="font-medium tabular-nums">{service.total}</span>
                    </div>
                  )}
                  {service.expired !== undefined && service.expired > 0 && (
                    <div className="flex justify-between">
                      <span className="text-slate-500">Expired</span>
                      <span className="font-medium tabular-nums text-amber-600">{service.expired}</span>
                    </div>
                  )}
                  {service.sync_errors !== undefined && service.sync_errors > 0 && (
                    <div className="flex justify-between">
                      <span className="text-slate-500">Sync Errors</span>
                      <span className="font-medium tabular-nums text-red-600">{service.sync_errors}</span>
                    </div>
                  )}
                  {service.with_issues !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-slate-500">With Issues</span>
                      <span className={cn("font-medium tabular-nums", service.with_issues > 0 ? "text-amber-600" : "")}>
                        {service.with_issues}
                      </span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
