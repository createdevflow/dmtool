export function auditedHealthScore(score: unknown): number | null {
  const n = Number(score);
  if (!Number.isFinite(n) || n <= 0) return null;
  return Math.round(n);
}

export function formatHealthOver100(score: unknown): string {
  const n = auditedHealthScore(score);
  return n === null ? "—" : `${n}/100`;
}

export function formatHealthPercent(score: unknown): string {
  const n = auditedHealthScore(score);
  return n === null ? "—" : `${n}%`;
}
