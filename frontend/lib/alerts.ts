export type StoredAlert = {
  id: string;
  kind?: string;
  title: string;
  body?: string;
  href?: string;
  severity?: string;
  is_read?: boolean;
  count?: number;
};

export function parseAlertsPayload(raw: unknown): StoredAlert[] {
  const envelope = raw as { data?: unknown } | null;
  const data = envelope && typeof envelope === "object" && "data" in envelope ? envelope.data : raw;
  if (Array.isArray(data)) {
    return data as StoredAlert[];
  }
  const obj = (data ?? {}) as { alerts?: unknown };
  return Array.isArray(obj.alerts) ? (obj.alerts as StoredAlert[]) : [];
}

export function unreadAlertCount(alerts: StoredAlert[]): number {
  return alerts.filter((a) => a.is_read !== true).length;
}
