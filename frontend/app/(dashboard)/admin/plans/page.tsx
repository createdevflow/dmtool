"use client";

import { useEffect, useState } from "react";
import { CreditCard, Plus, Pencil, Loader2 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { adminApi, type AdminPlan } from "@/lib/api-client";

// Form values for both create + edit. Edit pre-fills from the row;
// create starts with a sensible default. `code` is locked on edit
// (changing a plan's primary key is a footgun — users on that
// plan_code would have their entitlements silently break).
type PlanFormValues = {
  code: string;
  name: string;
  description: string;
  tier_rank: number;
  monthly_cents: number;
  yearly_cents: number;
  max_sites: number;
  is_active: boolean;
};

const emptyValues: PlanFormValues = {
  code: "",
  name: "",
  description: "",
  tier_rank: 0,
  monthly_cents: 0,
  yearly_cents: 0,
  max_sites: 1,
  is_active: true,
};

function fromPlan(p: AdminPlan): PlanFormValues {
  return {
    code: p.code,
    name: p.name,
    description: p.description ?? "",
    tier_rank: p.tier_rank,
    monthly_cents: p.monthly_cents,
    yearly_cents: p.yearly_cents,
    max_sites: p.max_sites,
    is_active: p.is_active,
  };
}

// PlanModal is a tiny self-contained modal for create/edit. The
// implementation plan calls for a shared Dialog/Sheet component in
// the polish phase; this is one usage so we don't ship a generic
// primitive yet. When phase 8 lands the shared Dialog, the markup
// here is the only thing that has to move.
//
// The plan's row id is passed as a sibling prop (planId) — not
// smuggled through the form values — so the form values describe
// the plan's editable content, not its identity.
function PlanModal({
  mode,
  planId,
  initial,
  onClose,
  onSaved,
}: {
  mode: "create" | "edit";
  planId: number;
  initial: PlanFormValues;
  onClose: () => void;
  onSaved: (saved: AdminPlan) => void;
}) {
  const [values, setValues] = useState<PlanFormValues>(initial);
  const [busy, setBusy] = useState(false);
  const isEdit = mode === "edit";

  const set = <K extends keyof PlanFormValues>(k: K, v: PlanFormValues[K]) => {
    setValues((prev) => ({ ...prev, [k]: v }));
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!values.code.trim() || !values.name.trim()) {
      toast("Code and name are required", "error");
      return;
    }
    if (!/^[a-z][a-z0-9_]{1,40}$/.test(values.code)) {
      toast(
        "Code must be lowercase letters, digits, and underscores, starting with a letter",
        "error"
      );
      return;
    }
    if (values.monthly_cents < 0 || values.yearly_cents < 0) {
      toast("Prices must be non-negative (cents)", "error");
      return;
    }
    if (values.max_sites < 1) {
      toast("Max sites must be at least 1", "error");
      return;
    }
    setBusy(true);
    try {
      let r;
      if (isEdit) {
        r = await adminApi.updatePlan(planId, {
          name: values.name,
          description: values.description,
          tier_rank: values.tier_rank,
          monthly_cents: values.monthly_cents,
          yearly_cents: values.yearly_cents,
          max_sites: values.max_sites,
          is_active: values.is_active,
        });
      } else {
        r = await adminApi.createPlan({
          code: values.code,
          name: values.name,
          description: values.description,
          tier_rank: values.tier_rank,
          monthly_cents: values.monthly_cents,
          yearly_cents: values.yearly_cents,
          max_sites: values.max_sites,
        });
      }
      onSaved(r.data.data);
      toast(isEdit ? "Plan updated" : "Plan created", "success");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Save failed";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={isEdit ? "Edit plan" : "Create plan"}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-slate-900 mb-4">
          {isEdit ? `Edit plan: ${initial.code}` : "Create plan"}
        </h2>
        <form onSubmit={submit} className="space-y-3">
          <div>
            <Label htmlFor="plan-code">Code</Label>
            <Input
              id="plan-code"
              value={values.code}
              onChange={(e) => set("code", e.target.value)}
              disabled={isEdit}
              placeholder="e.g. team_monthly"
              autoFocus
            />
            {isEdit && (
              <p className="text-xs text-slate-500 mt-1">
                Code is the plan&apos;s primary key. It can&apos;t be
                changed after creation.
              </p>
            )}
          </div>
          <div>
            <Label htmlFor="plan-name">Name</Label>
            <Input
              id="plan-name"
              value={values.name}
              onChange={(e) => set("name", e.target.value)}
              placeholder="e.g. Team · Monthly"
            />
          </div>
          <div>
            <Label htmlFor="plan-desc">Description</Label>
            <Textarea
              id="plan-desc"
              value={values.description}
              onChange={(e) => set("description", e.target.value)}
              rows={2}
              placeholder="Short marketing line shown on /pricing."
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label htmlFor="plan-tier">Tier rank</Label>
              <Input
                id="plan-tier"
                type="number"
                value={values.tier_rank}
                onChange={(e) => set("tier_rank", Number(e.target.value))}
              />
              <p className="text-xs text-slate-500 mt-1">Higher = ranked above.</p>
            </div>
            <div>
              <Label htmlFor="plan-max">Max sites</Label>
              <Input
                id="plan-max"
                type="number"
                min={1}
                value={values.max_sites}
                onChange={(e) => set("max_sites", Number(e.target.value))}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label htmlFor="plan-monthly">Monthly (cents)</Label>
              <Input
                id="plan-monthly"
                type="number"
                min={0}
                value={values.monthly_cents}
                onChange={(e) => set("monthly_cents", Number(e.target.value))}
              />
              <p className="text-xs text-slate-500 mt-1">
                = ${(values.monthly_cents / 100).toFixed(2)}/mo
              </p>
            </div>
            <div>
              <Label htmlFor="plan-yearly">Yearly (cents)</Label>
              <Input
                id="plan-yearly"
                type="number"
                min={0}
                value={values.yearly_cents}
                onChange={(e) => set("yearly_cents", Number(e.target.value))}
              />
              <p className="text-xs text-slate-500 mt-1">
                = ${(values.yearly_cents / 100).toFixed(2)}/yr
              </p>
            </div>
          </div>
          {isEdit && (
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={values.is_active}
                onChange={(e) => set("is_active", e.target.checked)}
              />
              Active
            </label>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
              Cancel
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
              {isEdit ? "Save" : "Create"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function AdminPlansPage() {
  const [plans, setPlans] = useState<AdminPlan[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<
    | { kind: "create" }
    | { kind: "edit"; plan: AdminPlan; values: PlanFormValues }
    | null
  >(null);

  const refresh = async () => {
    setLoading(true);
    try {
      const r = await adminApi.listPlans();
      setPlans(r.data.data);
    } catch {
      // Guard handles 403/401.
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const fmtCents = (cents: number) =>
    cents === 0 ? "—" : `$${(cents / 100).toFixed(0)}`;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <CreditCard className="w-6 h-6 text-slate-700" />
            Plans
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {plans.length} plan{plans.length === 1 ? "" : "s"} configured.
            Edit prices and limits; new tiers ship without code changes.
          </p>
        </div>
        <Button onClick={() => setEditing({ kind: "create" })}>
          <Plus className="w-4 h-4 mr-1" />
          New plan
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
                  <th className="px-4 py-3 font-medium">Tier</th>
                  <th className="px-4 py-3 font-medium">Monthly</th>
                  <th className="px-4 py-3 font-medium">Yearly</th>
                  <th className="px-4 py-3 font-medium">Max sites</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {plans.map((p) => (
                  <tr key={p.id} className="border-b border-slate-100">
                    <td className="px-4 py-3 font-mono text-xs text-slate-900">{p.code}</td>
                    <td className="px-4 py-3 text-slate-700">
                      <div className="font-medium text-slate-900">{p.name}</div>
                      <div className="text-xs text-slate-500">{p.description || "—"}</div>
                    </td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{p.tier_rank}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{fmtCents(p.monthly_cents)}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{fmtCents(p.yearly_cents)}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{p.max_sites}</td>
                    <td className="px-4 py-3">
                      <Badge
                        className={
                          p.is_active
                            ? "bg-emerald-100 text-emerald-800 hover:bg-emerald-100"
                            : "bg-slate-200 text-slate-700 hover:bg-slate-200"
                        }
                      >
                        {p.is_active ? "active" : "inactive"}
                      </Badge>
                    </td>
                    <td className="px-2 py-3">
                      <button
                        onClick={() =>
                          setEditing({ kind: "edit", plan: p, values: fromPlan(p) })
                        }
                        className="text-slate-400 hover:text-slate-700 inline-flex items-center gap-1 text-xs"
                      >
                        <Pencil className="w-3 h-3" /> Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {editing?.kind === "create" && (
        <PlanModal
          mode="create"
          planId={0}
          initial={emptyValues}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            refresh();
          }}
        />
      )}
      {editing?.kind === "edit" && (
        <PlanModal
          mode="edit"
          planId={editing.plan.id}
          initial={editing.values}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            refresh();
          }}
        />
      )}
    </div>
  );
}
