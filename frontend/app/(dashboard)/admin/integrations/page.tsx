"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Blocks, ArrowLeft, CheckCircle, AlertTriangle, XCircle, RefreshCw
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { adminApi } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";

type IntegrationsData = {
  total_connections: number;
  active_connections: number;
  unique_users: number;
  unique_projects: number;
  providers: Array<{
    provider: string;
    total: number;
    expired: number;
    sync_errors: number;
    last_synced_at: string | null;
  }>;
};

const PROVIDER_LABELS: Record<string, string> = {
  google: "Google Search Console",
  meta: "Meta (Instagram/Facebook)",
  linkedin: "LinkedIn",
};

const PROVIDER_ICONS: Record<string, string> = {
  google: "G",
  meta: "M",
  linkedin: "L",
};

export default function AdminIntegrationsPage() {
  const [data, setData] = useState<IntegrationsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchData = async (showRefresh = false) => {
    if (showRefresh) setRefreshing(true);
    try {
      const r = await adminApi.integrationsStatus();
      setData(r.data.data);
    } catch {
      // Non-critical.
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  if (loading) {
    return <div className="text-slate-400 py-12 text-center">Loading integration status…</div>;
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
            <Blocks className="w-6 h-6 text-slate-700" />
            Integration Status
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            OAuth provider connections across the platform.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchData(true)}
          disabled={refreshing}
        >
          <RefreshCw className={cn("w-4 h-4 mr-2", refreshing && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">Total Connections</div>
            <div className="text-2xl font-bold tabular-nums text-slate-900">{data?.total_connections || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">Active</div>
            <div className="text-2xl font-bold tabular-nums text-emerald-600">{data?.active_connections || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">Users Connected</div>
            <div className="text-2xl font-bold tabular-nums text-slate-900">{data?.unique_users || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">Projects Connected</div>
            <div className="text-2xl font-bold tabular-nums text-slate-900">{data?.unique_projects || 0}</div>
          </CardContent>
        </Card>
      </div>

      {/* Provider cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {(data?.providers || []).length === 0 ? (
          <Card className="md:col-span-2 lg:col-span-3">
            <CardContent className="py-12 text-center text-slate-500">
              No integrations configured yet.
            </CardContent>
          </Card>
        ) : (
          (data?.providers || []).map((p) => {
            const hasIssues = p.expired > 0 || p.sync_errors > 0;
            return (
              <Card key={p.provider}>
                <CardContent className="p-5">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-slate-100 flex items-center justify-center text-sm font-bold text-slate-600">
                        {PROVIDER_ICONS[p.provider] || "?"}
                      </div>
                      <div>
                        <div className="font-medium text-slate-900">
                          {PROVIDER_LABELS[p.provider] || p.provider}
                        </div>
                        <Badge className={cn("text-[10px] mt-1", hasIssues ? "bg-amber-100 text-amber-800" : "bg-emerald-100 text-emerald-800")}>
                          {hasIssues ? "Issues" : "Healthy"}
                        </Badge>
                      </div>
                    </div>
                    {hasIssues ? (
                      <AlertTriangle className="w-5 h-5 text-amber-500" />
                    ) : (
                      <CheckCircle className="w-5 h-5 text-emerald-500" />
                    )}
                  </div>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-slate-500">Connections</span>
                      <span className="font-medium tabular-nums">{p.total}</span>
                    </div>
                    {p.expired > 0 && (
                      <div className="flex justify-between">
                        <span className="text-slate-500">Expired</span>
                        <span className="font-medium tabular-nums text-amber-600">{p.expired}</span>
                      </div>
                    )}
                    {p.sync_errors > 0 && (
                      <div className="flex justify-between">
                        <span className="text-slate-500">Sync Errors</span>
                        <span className="font-medium tabular-nums text-red-600">{p.sync_errors}</span>
                      </div>
                    )}
                    <div className="flex justify-between">
                      <span className="text-slate-500">Last Sync</span>
                      <span className="font-medium text-xs">
                        {p.last_synced_at ? formatDate(p.last_synced_at) : "Never"}
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            );
          })
        )}
      </div>
    </div>
  );
}
