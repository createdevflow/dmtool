"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { Search, Users, ChevronRight, Download, Filter, X } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { adminApi, type AdminUserSummary } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { toast } from "@/components/ui/toaster";

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUserSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [role, setRole] = useState("");
  const [mode, setMode] = useState("");
  const [status, setStatus] = useState("");
  const [plan, setPlan] = useState("");
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);
  const size = 25;

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const r = await adminApi.listUsers({
        page,
        size,
        search: search || undefined,
        role: role || undefined,
        mode: mode || undefined,
        status: status || undefined,
        plan: plan || undefined,
      });
      setUsers(r.data.data.users);
      setTotal(r.data.data.total);
    } catch {
      // Guard will redirect on 403.
    } finally {
      setLoading(false);
    }
  }, [page, search, role, mode, status, plan]);

  useEffect(() => {
    let cancelled = false;
    fetchUsers();
    return () => { cancelled = true; };
  }, [fetchUsers]);

  const handleExport = async () => {
    try {
      const r = await adminApi.exportUsers();
      const blob = new Blob([r.data as any], { type: "text/csv" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "users_export.csv";
      a.click();
      URL.revokeObjectURL(url);
      toast("Users exported", "success");
    } catch {
      toast("Export failed", "error");
    }
  };

  const hasActiveFilters = role || mode || status || plan;
  const clearFilters = () => {
    setRole("");
    setMode("");
    setStatus("");
    setPlan("");
    setPage(1);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Users className="w-6 h-6 text-slate-700" />
            Users
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {total.toLocaleString()} total user{total === 1 ? "" : "s"}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={handleExport}>
          <Download className="w-4 h-4 mr-2" />
          Export CSV
        </Button>
      </div>

      {/* Search + Filter toggle */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <Input
            placeholder="Search by email or name"
            value={search}
            onChange={(e) => {
              setPage(1);
              setSearch(e.target.value);
            }}
            className="pl-9"
          />
        </div>
        <Button
          variant={showFilters ? "default" : "outline"}
          size="sm"
          onClick={() => setShowFilters(!showFilters)}
          className="relative"
        >
          <Filter className="w-4 h-4 mr-2" />
          Filters
          {hasActiveFilters && (
            <span className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-brand-600 text-white text-[10px] flex items-center justify-center">
              {[role, mode, status, plan].filter(Boolean).length}
            </span>
          )}
        </Button>
        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            <X className="w-4 h-4 mr-1" />
            Clear
          </Button>
        )}
      </div>

      {/* Filter dropdowns */}
      {showFilters && (
        <Card>
          <CardContent className="p-4">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div>
                <label className="text-xs font-medium text-slate-500 uppercase tracking-wide">Role</label>
                <select
                  value={role}
                  onChange={(e) => { setRole(e.target.value); setPage(1); }}
                  className="mt-1 block w-full rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm"
                >
                  <option value="">All roles</option>
                  <option value="owner">Owner</option>
                  <option value="admin">Admin</option>
                  <option value="viewer">Viewer</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-slate-500 uppercase tracking-wide">Mode</label>
                <select
                  value={mode}
                  onChange={(e) => { setMode(e.target.value); setPage(1); }}
                  className="mt-1 block w-full rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm"
                >
                  <option value="">All modes</option>
                  <option value="combined">Combined</option>
                  <option value="search">SEO</option>
                  <option value="social">Social</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-slate-500 uppercase tracking-wide">Status</label>
                <select
                  value={status}
                  onChange={(e) => { setStatus(e.target.value); setPage(1); }}
                  className="mt-1 block w-full rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm"
                >
                  <option value="">All statuses</option>
                  <option value="active">Active</option>
                  <option value="suspended">Suspended</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-slate-500 uppercase tracking-wide">Plan</label>
                <select
                  value={plan}
                  onChange={(e) => { setPlan(e.target.value); setPage(1); }}
                  className="mt-1 block w-full rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm"
                >
                  <option value="">All plans</option>
                  <option value="free">Free</option>
                  <option value="pro_monthly">Pro Monthly</option>
                  <option value="pro_yearly">Pro Yearly</option>
                </select>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-400">Loading…</div>
          ) : users.length === 0 ? (
            <div className="p-8 text-center text-slate-500">No users match your filters.</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-200">
                  <th className="px-4 py-3 font-medium">Email</th>
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">Role</th>
                  <th className="px-4 py-3 font-medium">Plan</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium">Joined</th>
                  <th className="px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} className="border-b border-slate-100 hover:bg-slate-50">
                    <td className="px-4 py-3 font-medium text-slate-900">{u.email}</td>
                    <td className="px-4 py-3 text-slate-700">{u.name || "—"}</td>
                    <td className="px-4 py-3">
                      <Badge
                        variant="outline"
                        className={
                          u.role === "admin"
                            ? "border-purple-300 text-purple-800"
                            : "text-slate-600"
                        }
                      >
                        {u.role}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-slate-700">{u.plan_code}</td>
                    <td className="px-4 py-3">
                      <Badge
                        className={
                          u.sub_status === "active"
                            ? "bg-emerald-100 text-emerald-800 hover:bg-emerald-100"
                            : u.sub_status === "trialing"
                            ? "bg-blue-100 text-blue-800 hover:bg-blue-100"
                            : "bg-slate-200 text-slate-700 hover:bg-slate-200"
                        }
                      >
                        {u.sub_status}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-slate-500 tabular-nums">{formatDate(u.created_at)}</td>
                    <td className="px-2 py-3">
                      <Link
                        href={`/admin/users/${u.id}`}
                        className="text-slate-400 hover:text-slate-700 inline-flex"
                      >
                        <ChevronRight className="w-4 h-4" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {total > size && (
        <div className="flex items-center justify-between text-sm text-slate-500">
          <span>
            Showing {(page - 1) * size + 1}–{Math.min(page * size, total)} of {total}
          </span>
          <div className="flex gap-2">
            <button
              disabled={page === 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="px-3 py-1 rounded-md border border-slate-200 disabled:opacity-50"
            >
              Prev
            </button>
            <button
              disabled={page * size >= total}
              onClick={() => setPage((p) => p + 1)}
              className="px-3 py-1 rounded-md border border-slate-200 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
