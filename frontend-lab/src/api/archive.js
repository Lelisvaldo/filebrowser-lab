// Compactados como pastas (7-Zip no servidor).
import { get, send, apiPath, request } from "./client";
import { baseURL } from "@/lib/config";

const hdr = (pw) => (pw ? { "X-Archive-Password": encodeURIComponent(pw) } : {});
export const list = (source, path, dir, pw) => get("lab/archive/list", { source, path, dir }, { headers: hdr(pw) });
export const fileUrl = (source, path, inner, download) => apiPath("lab/archive/file", { source, path, inner, download: download ? "1" : undefined });
export async function fileBlob(source, path, inner, pw) {
  const res = await request("lab/archive/file", { params: { source, path, inner }, headers: hdr(pw), raw: true });
  return res.blob();
}
export const edit = (body, pw) => send("lab/archive/edit", "POST", body, undefined, { headers: hdr(pw) });
export const create = (body) => send("lab/archive/create", "POST", body);
export const extract = (body) => send("lab/archive/extract", "POST", body);
export async function add(source, path, dir, files, pw) {
  const fd = new FormData();
  fd.append("source", source); fd.append("path", path); fd.append("dir", dir);
  for (const f of files) { fd.append("rel", f.rel || f.file.name); fd.append("file", f.file, f.file.name); }
  const res = await fetch(`${baseURL}api/lab/archive/add`, { method: "POST", body: fd, credentials: "same-origin", headers: hdr(pw) });
  if (!res.ok) { let m = res.statusText; try { const j = await res.json(); m = j.message || j.error || m; } catch (_e) { /* texto */ } throw new Error(m); }
  return res.json();
}
