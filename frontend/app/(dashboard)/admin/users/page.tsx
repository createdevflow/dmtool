"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Search, Users, ChevronRight } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { adminApi, type AdminUserSummary } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUserSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const size = 25;

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    adminApi
      .listUsers({ page, size, search: search || undefined })
      .then((r) => {
        if (cancelled) return;
        setUsers(r.data.data.users);
        setTotal(r.data.data.total);
      })
      .catch(() => {
        // Guard will redirect on 403.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [page, search]);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <Users className="w-6 h-6 text-slate-700" />
          Users
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          {total.toLocaleString()} total user{total === 1 ? "" : "s"}
        </p>
      </div>

      <div className="relative max-w-sm">
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

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-400">Loading…</div>
          ) : users.length === 0 ? (
            <div className="p-8 text-center text-slate-500">No users match your search.</div>
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
