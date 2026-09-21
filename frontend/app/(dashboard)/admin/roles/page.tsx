"use client";

import { useEffect, useMemo, useState } from "react";
import { Plus, Pencil, Shield, Trash2 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  adminApi,
  type AdminPermission,
  type AdminRole,
} from "@/lib/api-client";

type RoleForm = {
  code: string;
  name: string;
  description: string;
  permissions: string[];
};

const emptyForm: RoleForm = {
  code: "",
  name: "",
  description: "",
  permissions: ["admin.access"],
};

function apiError(e: unknown): string {
  if (typeof e === "object" && e !== null && "response" in e) {
    const data = (e as { response?: { data?: { error?: { message?: string } } } }).response?.data;
    const msg = data?.error?.message;
    if (msg) return msg;
  }
  if (e instanceof Error) return e.message;
  return "Request failed";
}

function RoleModal({
  mode,
  role,
  catalog,
  onClose,
  onSaved,
}: {
  mode: "create" | "edit" | "view";
  role: AdminRole | null;
  catalog: AdminPermission[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const readOnly = mode === "view";
  const [values, setValues] = useState<RoleForm>(() => {
    if (!role) return emptyForm;
    return {
      code: role.code,
      name: role.name,
      description: role.description ?? "",
      permissions: (role.permissions || []).map((p) => p.code),
    };
  });
  const [busy, setBusy] = useState(false);

  const grouped = useMemo(() => {
    const map = new Map<string, AdminPermission[]>();
    for (const p of catalog) {
      const list = map.get(p.category) || [];
      list.push(p);
      map.set(p.category, list);
    }
    return Array.from(map.entries());
  }, [catalog]);

  const toggle = (code: string) => {
    if (readOnly || code === "admin.access") return;
    setValues((prev) => {
      const has = prev.permissions.includes(code);
      return {
        ...prev,
        permissions: has
          ? prev.permissions.filter((c) => c !== code)
          : [...prev.permissions, code],
      };
    });
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (readOnly) return;
    if (!values.name.trim() || (mode === "create" && !values.code.trim())) {
      toast("Code and name are required", "error");
      return;
    }
    if (mode === "create" && !/^[a-z][a-z0-9_]{1,40}$/.test(values.code)) {
      toast(
        "Code must be lowercase letters, digits, and underscores, starting with a letter",
        "error"
      );
      return;
    }
    if (!values.permissions.includes("admin.access")) {
      toast("Staff roles must include admin.access", "error");
      return;
    }
    setBusy(true);
    try {
      if (mode === "create") {
        await adminApi.createRole({
          code: values.code,
          name: values.name,
          description: values.description,
          permissions: values.permissions,
        });
        toast("Role created", "success");
      } else if (role) {
        await adminApi.updateRole(role.id, {
          name: values.name,
          description: values.description,
          permissions: values.permissions,
        });
        toast("Role updated", "success");
      }
      onSaved();
    } catch (err) {
      toast(apiError(err), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent
        aria-label={mode === "create" ? "Create role" : `Role ${values.name}`}
        maxWidthClass="max-w-2xl"
        hideClose
      >
        <DialogHeader>
          <DialogTitle>
            {mode === "create" && "Create role"}
            {mode === "edit" && `Edit role: ${role?.code}`}
            {mode === "view" && `Super Admin (system)`}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label htmlFor="role-code">Code</Label>
              <Input
                id="role-code"
                value={values.code}
                onChange={(e) => setValues((v) => ({ ...v, code: e.target.value }))}
                disabled={mode !== "create"}
                placeholder="e.g. support"
                autoFocus={mode === "create"}
              />
            </div>
            <div>
              <Label htmlFor="role-name">Name</Label>
              <Input
                id="role-name"
                value={values.name}
                onChange={(e) => setValues((v) => ({ ...v, name: e.target.value }))}
                disabled={readOnly}
                placeholder="e.g. Support"
              />
            </div>
          </div>
          <div>
            <Label htmlFor="role-desc">Description</Label>
            <Textarea
              id="role-desc"
              value={values.description}
              onChange={(e) => setValues((v) => ({ ...v, description: e.target.value }))}
              disabled={readOnly}
              rows={2}
            />
          </div>
          <div>
            <p className="text-xs font-medium text-slate-500 uppercase tracking-wide mb-2">
              Permissions
            </p>
            <div className="space-y-3 max-h-[40vh] overflow-y-auto pr-1">
              {grouped.map(([category, perms]) => (
                <div key={category}>
                  <p className="text-xs font-semibold text-slate-700 capitalize mb-1">{category}</p>
                  <div className="space-y-1">
                    {perms.map((p) => {
                      const checked = values.permissions.includes(p.code);
                      const locked = p.code === "admin.access";
                      return (
                        <label
                          key={p.code}
                          className="flex items-start gap-2 rounded-md px-2 py-1.5 hover:bg-slate-50"
                        >
                          <input
                            type="checkbox"
                            className="mt-1"
                            checked={checked}
                            disabled={readOnly || locked}
                            onChange={() => toggle(p.code)}
                          />
                          <span>
                            <span className="block text-sm text-slate-900">{p.name}</span>
                            <span className="block text-xs text-slate-500 font-mono">{p.code}</span>
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
              {readOnly ? "Close" : "Cancel"}
            </Button>
            {!readOnly && (
              <Button type="submit" disabled={busy}>
                {mode === "create" ? "Create" : "Save"}
              </Button>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default function AdminRolesPage() {
  const [roles, setRoles] = useState<AdminRole[]>([]);
  const [catalog, setCatalog] = useState<AdminPermission[]>([]);
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState<
    | { kind: "create" }
    | { kind: "edit"; role: AdminRole }
    | { kind: "view"; role: AdminRole }
    | null
  >(null);

  const refresh = async () => {
    setLoading(true);
    try {
      const [rr, pr] = await Promise.all([adminApi.listRoles(), adminApi.listPermissions()]);
      setRoles(rr.data.data || []);
      setCatalog(pr.data.data || []);
    } catch {
      // Guard handles 403.
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const handleDelete = async (role: AdminRole) => {
    if (role.is_system) return;
    if (!confirm(`Delete role ${role.code}? This cannot be undone.`)) return;
    try {
      await adminApi.deleteRole(role.id);
      toast("Role deleted", "success");
      await refresh();
    } catch (err) {
      toast(apiError(err), "error");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Shield className="w-6 h-6 text-slate-700" />
            Roles
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            Staff roles and the permission catalog. Super Admin is a system role and cannot be edited.
            Permission codes are developer-maintained — this page only assigns them.
          </p>
        </div>
        <Button onClick={() => setModal({ kind: "create" })}>
          <Plus className="w-4 h-4 mr-1" />
          New role
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-400">Loading…</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-200">
                  <th className="px-4 py-3 font-medium">Code</th>
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">Permissions</th>
                  <th className="px-4 py-3 font-medium">Users</th>
                  <th className="px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {roles.map((role) => (
                  <tr key={role.id} className="border-b border-slate-100">
                    <td className="px-4 py-3 font-mono text-xs text-slate-900">{role.code}</td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-slate-900 flex items-center gap-2">
                        {role.name}
                        {role.is_system && (
                          <Badge variant="outline" className="border-purple-300 text-purple-800">
                            system
                          </Badge>
                        )}
                      </div>
                      <div className="text-xs text-slate-500">{role.description || "—"}</div>
                    </td>
                    <td className="px-4 py-3 tabular-nums text-slate-700">
                      {(role.permissions || []).length}
                    </td>
                    <td className="px-4 py-3 tabular-nums text-slate-700">
                      {role.user_count ?? 0}
                    </td>
                    <td className="px-2 py-3 text-right whitespace-nowrap">
                      {role.is_system ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setModal({ kind: "view", role })}
                        >
                          View
                        </Button>
                      ) : (
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setModal({ kind: "edit", role })}
                          >
                            <Pencil className="w-3.5 h-3.5 mr-1" />
                            Edit
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            className="text-red-700 border-red-200 hover:bg-red-50"
                            onClick={() => handleDelete(role)}
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </Button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {modal?.kind === "create" && (
        <RoleModal mode="create" role={null} catalog={catalog} onClose={() => setModal(null)} onSaved={() => { setModal(null); refresh(); }} />
      )}
      {modal?.kind === "edit" && (
        <RoleModal mode="edit" role={modal.role} catalog={catalog} onClose={() => setModal(null)} onSaved={() => { setModal(null); refresh(); }} />
      )}
      {modal?.kind === "view" && (
        <RoleModal mode="view" role={modal.role} catalog={catalog} onClose={() => setModal(null)} onSaved={() => setModal(null)} />
      )}
    </div>
  );
}
