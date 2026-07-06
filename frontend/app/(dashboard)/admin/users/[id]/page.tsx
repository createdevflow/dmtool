"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useRouter } from "next/navigation";
import { User, Mail, KeyRound, Eye, ArrowLeft, ShieldCheck, Calendar, Activity } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { adminApi, type AdminUserSummary, type AdminAuditEntry, type Subscription } from "@/lib/api-client";
import { formatDate, formatDateTime } from "@/lib/utils";
import { ImpersonationBanner } from "@/components/admin/impersonation-banner";

export default function AdminUserDetailPage() {
  const router = useRouter();
  const params = useParams<{ id: string }>();
  const id = Number(params.id);

  const [user, setUser] = useState<AdminUserSummary | null>(null);
  const [sub, setSub] = useState<Subscription | null>(null);
  const [audit, setAudit] = useState<AdminAuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<{ name: string; email: string; role: string } | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const r = await adminApi.getUser(id);
      setUser(r.data.data.user as unknown as AdminUserSummary);
      setSub(r.data.data.subscription);
      setAudit(r.data.data.audit_log);
    } catch {
      // Guard handles 403/401.
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (!isNaN(id)) refresh();
  }, [id, refresh]);

  if (loading || !user) {
    return <div className="text-slate-400 py-12 text-center">Loading user…</div>;
  }

  const handleEdit = () => {
    setEditing({ name: user.name, email: user.email, role: user.role });
  };
  const cancelEdit = () => setEditing(null);
  const saveEdit = async () => {
    if (!editing) return;
    setBusy(true);
    try {
      await adminApi.updateUser(id, editing);
      toast("User updated", "success");
      setEditing(null);
      await refresh();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Update failed";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  const handleResetPassword = async () => {
    if (!confirm(`Reset password for ${user.email}?`)) return;
    setBusy(true);
    try {
      const r = await adminApi.resetPassword(id);
      const tp = r.data.data.temporary_password;
      // Show in a toast (not an alert) so the admin sees it once
      // and can copy it. We don't render it on the page because
      // rendering the plaintext would be an XSS / shoulder-surf
      // surface if the page is left open.
      toast(`Temp password: ${tp}`, "success");
      await refresh();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Reset failed";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  const handleImpersonate = async () => {
    if (!confirm(`Start impersonating ${user.email}? Your next requests will run as them.`)) return;
    setBusy(true);
    try {
      const r = await adminApi.impersonate(id);
      const token = r.data.data.token;
      const target = r.data.data.target;
      // Persist the impersonation token so the banner + subsequent
      // requests use it. LocalStorage is fine here because the
      // banner is a single-tab UI; a real prod setup would use a
      // secure cookie. Keep this out of the auth cookie path —
      // cookie swap is the admin's real token.
      localStorage.setItem("dmtool_admin_impersonation", JSON.stringify({
        token,
        target,
        expires_at: Date.now() + r.data.data.expires_in_minutes * 60 * 1000,
      }));
      toast(`Impersonating ${target.email}`, "info");
      await refresh();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Impersonation failed";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <ImpersonationBanner />

      <button
        onClick={() => router.push("/admin/users")}
        className="text-xs text-slate-500 hover:text-slate-900 inline-flex items-center gap-1"
      >
        <ArrowLeft className="w-3 h-3" /> Back to users
      </button>

      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <User className="w-6 h-6 text-slate-700" />
          {user.email}
        </h1>
        <p className="text-sm text-slate-500 mt-1">User ID #{user.id}</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Profile */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Profile</CardTitle>
            <CardDescription>Mutable fields the admin can edit.</CardDescription>
          </CardHeader>
          <CardContent>
            {editing ? (
              <div className="space-y-4">
                <div>
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    value={editing.name}
                    onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                  />
                </div>
                <div>
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    value={editing.email}
                    onChange={(e) => setEditing({ ...editing, email: e.target.value })}
                  />
                </div>
                <div>
                  <Label htmlFor="role">Role</Label>
                  <select
                    id="role"
                    value={editing.role}
                    onChange={(e) => setEditing({ ...editing, role: e.target.value })}
                    className="block w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm"
                  >
                    <option value="owner">owner</option>
                    <option value="admin">admin</option>
                    <option value="viewer">viewer</option>
                  </select>
                </div>
                <div className="flex gap-2">
                  <Button disabled={busy} onClick={saveEdit}>Save</Button>
                  <Button variant="outline" disabled={busy} onClick={cancelEdit}>Cancel</Button>
                </div>
              </div>
            ) : (
              <dl className="space-y-2 text-sm">
                <Row label="Name">{user.name || "—"}</Row>
                <Row label="Email"><Mail className="w-3 h-3 inline mr-1 text-slate-400" />{user.email}</Row>
                <Row label="Role">
                  <Badge variant="outline" className={user.role === "admin" ? "border-purple-300 text-purple-800" : "text-slate-600"}>
                    {user.role}
                  </Badge>
                </Row>
                <Row label="Plan">{user.plan_code}</Row>
                <Row label="Subscription status">
                  <Badge className={user.sub_status === "active" ? "bg-emerald-100 text-emerald-800" : "bg-slate-200 text-slate-700"}>
                    {user.sub_status}
                  </Badge>
                </Row>
                <Row label="Joined"><Calendar className="w-3 h-3 inline mr-1 text-slate-400" />{formatDate(user.created_at)}</Row>
                {user.trial_used_at && (
                  <Row label="Trial used at">{formatDate(user.trial_used_at)}</Row>
                )}
                <div className="pt-3">
                  <Button variant="outline" onClick={handleEdit} disabled={busy}>
                    Edit profile
                  </Button>
                </div>
              </dl>
            )}
          </CardContent>
        </Card>

        {/* Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Admin actions</CardTitle>
            <CardDescription>Audited. Use with care.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Button
              variant="outline"
              className="w-full justify-start"
              disabled={busy}
              onClick={handleResetPassword}
            >
              <KeyRound className="w-4 h-4 mr-2" />
              Reset password
            </Button>
            <Button
              variant="outline"
              className="w-full justify-start text-amber-700 border-amber-200 hover:bg-amber-50"
              disabled={busy}
              onClick={handleImpersonate}
            >
              <Eye className="w-4 h-4 mr-2" />
              Impersonate
            </Button>
            <p className="text-xs text-slate-500 pt-2">
              Both actions write an <ShieldCheck className="w-3 h-3 inline" /> audit-log row.
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Subscription */}
      {sub && (
        <Card>
          <CardHeader>
            <CardTitle>Subscription</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <Row label="Plan">{sub.plan_code}</Row>
              <Row label="Status">{sub.status}</Row>
              <Row label="Trial ends">{formatDate(sub.trial_ends_at)}</Row>
              <Row label="Started">{formatDate(sub.starts_at)}</Row>
            </dl>
          </CardContent>
        </Card>
      )}

      {/* Recent audit for this user */}
      <Card>
        <CardHeader>
          <CardTitle>
            <Activity className="w-4 h-4 inline mr-1" /> Recent admin activity (this user)
          </CardTitle>
        </CardHeader>
        <CardContent>
          {audit.length === 0 ? (
            <div className="text-sm text-slate-500">No admin actions against this user yet.</div>
          ) : (
            <ul className="space-y-2 text-xs">
              {audit.map((a) => (
                <li key={a.id} className="flex items-start gap-2">
                  <Badge variant="outline" className="text-[10px]">{a.action}</Badge>
                  <span className="text-slate-500 flex-1">
                    by admin {a.actor_user_id} · {formatDateTime(a.created_at)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline gap-3">
      <dt className="w-32 text-slate-500 text-xs uppercase tracking-wide">{label}</dt>
      <dd className="text-slate-900">{children}</dd>
    </div>
  );
}
