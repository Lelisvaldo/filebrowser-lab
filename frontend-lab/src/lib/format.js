// Formatação em pt-BR.
export function bytes(n, digits) {
  n = Number(n) || 0;
  const u = ["B", "KB", "MB", "GB", "TB", "PB"];
  let i = 0;
  while (n >= 1000 && i < u.length - 1) { n /= 1000; i++; }
  const d = digits !== undefined ? digits : (i === 0 ? 0 : n >= 100 || i < 2 ? 0 : n >= 10 ? 1 : 2);
  return n.toFixed(d).replace(".", ",") + " " + u[i];
}

export function speed(bps) {
  return bytes(bps, 0) + "/s";
}

export function duration(s) {
  if (!isFinite(s) || s < 0) return "";
  if (s < 60) return Math.max(1, Math.round(s)) + " s";
  if (s < 3600) return Math.round(s / 60) + " min";
  return (s / 3600).toFixed(1).replace(".", ",") + " h";
}

const fmtDate = new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short", year: "numeric" });
const fmtTime = new Intl.DateTimeFormat("pt-BR", { hour: "2-digit", minute: "2-digit" });

export function date(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d)) return "";
  const today = new Date();
  if (d.toDateString() === today.toDateString()) return "hoje " + fmtTime.format(d);
  return fmtDate.format(d).replace(/\./g, "");
}

export function pct(used, total) {
  return total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
}
