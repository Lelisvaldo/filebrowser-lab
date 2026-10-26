// API pública dos links compartilhados (não exige login).
import { baseURL } from "@/lib/config";

const url = (p, params = {}) => {
  const q = Object.entries(params).filter(([, v]) => v !== undefined && v !== "").map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`).join("&");
  return `${baseURL}public/api/${p}${q ? "?" + q : ""}`;
};
const headers = (pw) => (pw ? { "X-SHARE-PASSWORD": encodeURIComponent(pw) } : {});

async function check(res) {
  if (res.ok) return res;
  let msg = res.statusText;
  try { msg = (await res.json()).message || msg; } catch (_e) { /* texto */ }
  const e = new Error(msg);
  e.status = res.status;
  throw e;
}

export async function info(hash) {
  return (await check(await fetch(url("share/info", { hash }), { credentials: "same-origin" }))).json();
}

export async function list(hash, path, pw) {
  const d = await (await check(await fetch(url("resources", { hash, path }), { credentials: "same-origin", headers: headers(pw) }))).json();
  const base = d.path === "/" ? "/" : d.path;
  const items = [...(d.folders || []).map((f) => ({ ...f, type: "directory" })), ...(d.files || [])].map((i) => ({
    ...i, path: (base.endsWith("/") ? base : base + "/") + i.name + (i.type === "directory" ? "/" : ""),
  }));
  return { ...d, items };
}

export const downloadUrl = (hash, paths, archive) => url("resources/download", { hash, file: paths.length === 1 && paths[0] !== "/" ? paths[0] : undefined, algo: archive ? "zip" : undefined });

// Baixa com a senha do link (cabeçalho) e entrega ao navegador como arquivo.
export async function downloadWithPassword(hash, path, name, pw) {
  const res = await check(await fetch(downloadUrl(hash, [path]), { credentials: "same-origin", headers: headers(pw) }));
  const blob = await res.blob();
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 30000);
}
