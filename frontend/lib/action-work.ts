export type ActionWorkItem = {
  id: string;
  kind?: string;
  title: string;
  href?: string;
  done?: boolean;
  detail?: string;
  severity?: string;
};

export type ActionTask = {
  id: number;
  title: string;
  completed?: boolean;
  isDone?: boolean;
  source?: string;
  description?: string;
  priority?: string;
};

export function parseTasksPayload(raw: unknown): { tasks: ActionTask[]; work: ActionWorkItem[] } {
  const envelope = raw as { data?: unknown } | null;
  const data = envelope && typeof envelope === "object" && "data" in envelope ? envelope.data : raw;
  if (Array.isArray(data)) {
    return { tasks: data as ActionTask[], work: [] };
  }
  const obj = (data ?? {}) as { tasks?: unknown; work?: unknown };
  return {
    tasks: Array.isArray(obj.tasks) ? (obj.tasks as ActionTask[]) : [],
    work: Array.isArray(obj.work) ? (obj.work as ActionWorkItem[]) : [],
  };
}

export function isStoredTask(item: { id?: unknown }): boolean {
  return typeof item.id === "number" && item.id > 0;
}

export function formatTaskRow(t: ActionTask): ActionTask {
  return {
    ...t,
    isDone: t.isDone ?? !!t.completed,
    priority: t.priority || (t.source === "ai" ? "High" : "Medium"),
  };
}
