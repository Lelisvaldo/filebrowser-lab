// Cliente fino da API do Filebrowser Quantum (mesmo domínio, sessão por cookie HttpOnly).
import { baseURL } from "@/lib/config";

export class ApiError extends Error {
  constructor(status, message, body) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

let onUnauthorized = null;
export function setUnauthorizedHandler(fn) { onUnauthorized = fn; }

export function apiPath(path, params = {}) {
  const q = [];
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === "") continue;
    if (Array.isArray(v)) v.forEach((x) => q.push(`${encodeURIComponent(k)}=${encodeURIComponent(x)}`));
    else q.push(`${encodeURIComponent(k)}=${encodeURIComponent(v)}`);
  }
  return `${baseURL}api/${path.replace(/^\//, "")}${q.length ? "?" + q.join("&") : ""}`;
}

async function failure(res) {
  const text = await res.text();
  let msg = text;
  let body = null;
  try { body = JSON.parse(text); msg = body.message || body.error || text; } catch (_e) { /* texto puro */ }
  return new ApiError(res.status, msg || res.statusText, body);
}

export async function request(path, { params, method = "GET", body, headers = {}, raw = false, quiet401 = false } = {}) {
  const init = { method, credentials: "same-origin", headers: { ...headers } };
  if (body !== undefined) {
    if (typeof body === "string" || body instanceof Blob || body instanceof ArrayBuffer) init.body = body;
    else { init.body = JSON.stringify(body); init.headers["Content-Type"] = "application/json"; }
  }
  let res;
  try {
    res = await fetch(apiPath(path, params), init);
  } catch (_e) {
    throw new ApiError(0, "Não foi possível falar com o servidor.");
  }
  if (!res.ok) {
    if (res.status === 401 && !quiet401 && onUnauthorized) onUnauthorized();
    throw await failure(res);
  }
  if (raw) return res;
  const ct = res.headers.get("content-type") || "";
  return ct.includes("json") ? res.json() : res.text();
}

export const get = (path, params, opts) => request(path, { ...opts, params });
export const send = (path, method, body, params, opts) => request(path, { ...opts, method, body, params });
