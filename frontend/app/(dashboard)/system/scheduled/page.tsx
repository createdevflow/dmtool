"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { dashboardApi } from "@/lib/api-client";

export default function ScheduledPostsPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any | null>(null);
  const [tasks, setTasks] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<any | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchData(); }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const all = pRes.data?.data || [];
      setProjects(all);
      if (all.length > 0) {
        const last = all[all.length - 1];
        setProject(last);
        const cal = await dashboardApi.getCalendar(last.id);
        setTasks(cal.data?.data || []);
      }
    } catch (err) {
      console.error(err);
    } finally { setLoading(false); }
  };

  const startEdit = (t: any) => {
    setEditing({ ...t });
  };

  const saveEdit = async () => {
    if (!editing || !project) return;
    setSaving(true);
    try {
      await dashboardApi.updateCalendarEvent(editing.id, {
        project_id: project.id,
        title: editing.title,
        caption: editing.caption,
        platform: editing.platform,
        content_type: editing.content_type || editing.contentType,
        due_date: editing.due_date || editing.dueDate,
      });
      await fetchData();
      setEditing(null);
    } catch (err) {
      alert("Failed to save");
    } finally { setSaving(false); }
  };

  const deleteTask = async (t: any) => {
    if (!confirm("Delete scheduled post?")) return;
    try {
      await dashboardApi.deleteCalendarEvent(t.id);
      await fetchData();
    } catch (err) { alert("Delete failed"); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="space-y-6 p-6">
      <h1 className="text-2xl font-bold">Scheduled Posts</h1>
      <Card className="p-4">
        {tasks.length === 0 ? (
          <div className="text-sm text-slate-500">No scheduled posts found.</div>
        ) : (
          <div className="space-y-3">
            {tasks.map((t: any) => (
              <div key={t.id} className="flex items-center justify-between p-3 border rounded-md bg-white">
                <div>
                  <div className="text-sm font-semibold">{t.title}</div>
                  <div className="text-xs text-slate-500">{t.content_type || t.contentType} • {new Date(t.due_date || t.dueDate).toLocaleString()}</div>
                </div>
                <div className="flex items-center gap-2">
                  <Button size="sm" onClick={() => startEdit(t)}>Edit</Button>
                  <Button size="sm" variant="outline" className="text-rose-600" onClick={() => deleteTask(t)}>Delete</Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {editing && (
        <Card className="p-4">
          <h2 className="text-lg font-semibold">Edit Scheduled Post</h2>
          <div className="mt-3 grid gap-3">
            <input className="p-2 border rounded" value={editing.title} onChange={(e) => setEditing({ ...editing, title: e.target.value })} />
            <textarea className="p-2 border rounded" value={editing.caption || editing.caption} onChange={(e) => setEditing({ ...editing, caption: e.target.value })} />
            <input className="p-2 border rounded" type="datetime-local" value={editing.due_date ? new Date(editing.due_date).toISOString().slice(0,16) : (editing.dueDate ? new Date(editing.dueDate).toISOString().slice(0,16) : '')} onChange={(e) => setEditing({ ...editing, due_date: new Date(e.target.value).toISOString() })} />
            <div className="flex gap-2">
              <Button onClick={saveEdit} disabled={saving}>{saving ? 'Saving...' : 'Save'}</Button>
              <Button variant="ghost" onClick={() => setEditing(null)}>Cancel</Button>
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
