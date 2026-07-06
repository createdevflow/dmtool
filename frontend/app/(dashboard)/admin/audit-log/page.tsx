"use client";

import { useEffect, useState } from "react";
import { ScrollText } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { adminApi, type AdminAuditEntry } from "@/lib/api-client";
import { formatDateTime } from "@/lib/utils";

export default function AdminAuditLogPage() {
  const [entries, setEntries] = useState<AdminAuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [actionFilter, setActionFilter] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    adminApi
      .listAuditLog({ action: actionFilter || undefined, size: 100 })
      .then((r) => {
        if (cancelled) return;
        setEntries(r.data.data.entries);
        setTotal(r.data.data.total);
      })
      .catch(() => {
        // Guard handles 403/401.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [actionFilter]);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <ScrollText className="w-6 h-6 text-slate-700" />
          Audit log
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          {total.toLocaleString()} admin action{total === 1 ? "" : "s"} recorded.
          Append-only.
        </p>
      </div>

      <div className="max-w-sm">
        <Input
          placeholder="Filter by action (e.g. impersonate.start)"
          value={actionFilter}
          onChange={(e) => setActionFilter(e.target.value)}
        />
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-400">Loading…</div>
          ) : entries.length === 0 ? (
            <div className="p-8 text-center text-slate-500">No matching entries.</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-200">
                  <th className="px-4 py-3 font-medium">When</th>
                  <th className="px-4 py-3 font-medium">Action</th>
                  <th className="px-4 py-3 font-medium">Actor</th>
                  <th className="px-4 py-3 font-medium">Target</th>
                  <th className="px-4 py-3 font-medium">Metadata</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((e) => (
                  <tr key={e.id} className="border-b border-slate-100 align-top">
                    <td className="px-4 py-3 text-slate-500 tabular-nums whitespace-nowrap">
                      {formatDateTime(e.created_at)}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant="outline" className="text-[10px]">{e.action}</Badge>
                    </td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">#{e.actor_user_id}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">#{e.target_user_id}</td>
                    <td className="px-4 py-3 text-xs text-slate-500 font-mono">
                      {e.metadata ? JSON.stringify(e.metadata) : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
