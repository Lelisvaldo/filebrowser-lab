// Fila de envios com progresso real (XHR), em partes quando o arquivo é grande.
import { reactive } from "vue";
import { apiPath, request } from "@/api/client";
import * as filesApi from "@/api/files";
import { app, toast, toastError, setBusyCheck } from "./app";
import { moveReason } from "@/lib/moveReason";

export const transfers = reactive({ items: [] });

const MB = 1024 * 1024;
const MAX_PARALLEL = 3;
let seq = 0;
const listeners = new Set();
export const onTransferDone = (fn) => { listeners.add(fn); return () => listeners.delete(fn); };
const fire = (t) => listeners.forEach((fn) => { try { fn(t); } catch (_e) { /* ignora */ } });

const rand = () => Math.random().toString(36).slice(2) + Date.now().toString(36);

// files: [{ file: File, rel: "sub/pasta/arquivo.txt" }]  (rel é o caminho relativo dentro de dir)
export function enqueueUploads(source, dir, files, { overwrite = false } = {}) {
  for (const { file, rel } of files) {
    const path = dir.replace(/\/?$/, "/") + rel.replace(/^\/+/, "");
    transfers.items.push({
      id: ++seq, kind: "up", name: rel, source, dir, path, size: file.size, loaded: 0, base: 0,
      status: "queued", error: "", overwrite, speed: 0, session: rand(), _file: file, _xhr: null, _samples: [], _started: 0,
    });
  }
  pump();
}

// Registra um download iniciado pelo navegador (ele mostra o progresso próprio).
export function noteDownload(name, size) {
  transfers.items.push({ id: ++seq, kind: "down", name, source: "", dir: "", path: "", size: size || 0, loaded: size || 0, base: 0, status: "done", error: "", speed: 0, at: Date.now() });
  trim();
}

// Mover/copiar dentro do servidor: o PATCH é uma chamada só (sem bytes), então a fila mostra "em andamento" até terminar.
export async function trackedMoveCopy(items, action, opts) {
  const first = items[0] || {};
  const jobId = (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + Math.random().toString(16).slice(2)).replace(/-/g, "");
  const nome = (p) => String(p || "").replace(/\/+$/, "").split("/").pop();
  const dest = (first.toSource || "") + (first.to ? String(first.to).replace(/[^/]*\/?$/, "") : "");
  const raw = {
    id: ++seq, kind: action === "move" ? "move" : "copy", name: items.length === 1 ? nome(first.from) : items.length + " itens",
    source: first.fromSource || "", dir: dest, path: "", size: 0, loaded: 0, base: 0, status: "running", error: "", speed: 0,
    label: (action === "move" ? "Movendo" : "Copiando") + " para " + dest,
    fromDir: (first.fromSource || "") + (first.from ? String(first.from).replace(/[^/]*\/?$/, "") : ""), // de onde vem
    item: 0, items: items.length,
  };
  transfers.items.push(raw);
  const t = transfers.items[transfers.items.length - 1]; // proxy reativo: só ele atualiza a tela
  // o servidor conta os bytes do item em andamento; aqui só se pergunta a cada segundo (mover na mesma partição é instantâneo e nem aparece)
  let last = { at: Date.now(), done: 0, item: 0 };
  let polling = false;
  const poll = setInterval(async () => {
    if (polling || t.status !== "running") return;
    polling = true;
    try {
      const s = await filesApi.jobProgress(jobId);
      if (!s || t.status !== "running") return;
      const now = Date.now();
      if (s.item !== last.item) last = { at: now, done: 0, item: s.item }; // item novo: recomeça a velocidade
      if (now > last.at && s.done >= last.done) {
        const inst = ((s.done - last.done) * 1000) / (now - last.at);
        t.speed = t.speed > 0 && s.item === t.item ? t.speed * 0.6 + inst * 0.4 : inst; // média suave
      }
      last.at = now; last.done = s.done;
      t.size = s.total; t.loaded = s.done; t.item = s.item; t.items = s.items; t.curName = s.name;
    } finally { polling = false; }
  }, 1000);
  try {
    const res = await filesApi.moveCopy(items, action, { ...opts, job: jobId });
    const failed = res?.failed?.length || 0;
    // o servidor numera "nome (2)" quando o destino já tem um item com o mesmo nome: avisa qual ficou
    const base = (p) => String(p || "").replace(/\/+$/, "").split("/").pop();
    const renamed = (res?.succeeded || []).map((o) => {
      const want = items.find((i) => i.from === o.fromPath);
      return want && base(want.to) !== base(o.toPath) ? base(o.toPath) : "";
    }).filter(Boolean);
    if (renamed.length) toast("Já existia um item com esse nome no destino: salvo como " + renamed.join(", ") + ".", "info");
    if (failed) { t.status = "error"; t.error = moveReason(res.failed[0]?.message) || failed + " item(ns) falharam"; }
    else { t.status = "done"; t.label = (action === "move" ? "Movido" : "Copiado") + " para " + dest + (renamed.length ? " · salvo como " + renamed.join(", ") : ""); setTimeout(() => { t.hidden = true; }, 8000); }
    return res;
  } catch (e) {
    t.status = "error";
    let motivo = "";
    try { motivo = JSON.parse(e.message)?.failed?.[0]?.message || ""; } catch (_e) { motivo = String(e.message || ""); }
    t.error = moveReason(motivo) || "Falhou";
    throw e;
  } finally {
    clearInterval(poll);
    trim();
  }
}

function trim() {
  const done = transfers.items.filter((t) => ["done", "error", "canceled"].includes(t.status));
  if (done.length > 30) {
    const drop = new Set(done.slice(0, done.length - 30).map((t) => t.id));
    transfers.items = transfers.items.filter((t) => !drop.has(t.id));
  }
}

export const visible = () => transfers.items.filter((t) => !t.hidden && t.status !== "canceled" && (t.kind === "up" || t.kind === "move" || t.kind === "copy"));
export const active = () => transfers.items.filter((t) => ["queued", "running", "paused"].includes(t.status));
setBusyCheck(() => active().length > 0); // atividade em andamento impede o logout por inatividade
export const running = () => transfers.items.filter((t) => t.status === "running" && t.kind === "up");

function pump() {
  const room = MAX_PARALLEL - running().length;
  const next = transfers.items.filter((t) => t.status === "queued").slice(0, Math.max(0, room));
  next.forEach((t) => void run(t));
}

async function ensureDirs(t) {
  const parts = t.path.split("/").filter(Boolean);
  parts.pop();
  let acc = "/";
  const rel = t.dir.split("/").filter(Boolean).length;
  for (let i = 0; i < parts.length; i++) {
    acc += parts[i] + "/";
    if (i < rel) continue;
    try { await filesApi.mkdir(t.source, acc); } catch (_e) { /* já existe */ }
  }
}

// arquivo do navegador (Blob) ou arquivo do celular servido pelo app (file.native): devolve os bytes de [a, b)
async function bytesOf(file, a, b) {
  if (!file.native) return a === 0 && b >= file.size ? file : file.slice(a, b);
  const r = await fetch(file.native, { headers: { Range: `bytes=${a}-${b - 1}` }, cache: "no-store" });
  if (!r.ok && r.status !== 206) throw new Error("Não consegui ler o arquivo no celular.");
  return r.blob();
}

function sendPart(t, blob, offset, total, chunked) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    t._xhr = xhr;
    const params = { source: t.source, path: t.path, override: t.overwrite ? "true" : "false" };
    let url = apiPath("resources", params);
    if (chunked) url += `&_chunkSeq=${offset}`;
    xhr.open("POST", url, true);
    xhr.withCredentials = true;
    xhr.setRequestHeader("X-File-Upload-Session", t.session);
    xhr.setRequestHeader("X-File-Total-Size", String(total));
    if (chunked) xhr.setRequestHeader("X-File-Chunk-Offset", String(offset));
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) setLoaded(t, offset + e.loaded);
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) return resolve();
      let msg = xhr.responseText;
      try { msg = JSON.parse(msg).message || msg; } catch (_e) { /* texto */ }
      const err = new Error(xhr.status === 409 ? "Já existe um arquivo com esse nome." : msg || "Falha no envio");
      err.status = xhr.status;
      reject(err);
    };
    xhr.onerror = () => reject(new Error("Falha de rede durante o envio."));
    xhr.onabort = () => { const e = new Error("abort"); e.aborted = true; reject(e); };
    xhr.send(blob);
  });
}

function setLoaded(t, loaded) {
  const now = performance.now();
  t.loaded = Math.min(loaded, t.size);
  t._samples.push([now, t.loaded]);
  while (t._samples.length > 2 && now - t._samples[0][0] > 4000) t._samples.shift();
  const [t0, l0] = t._samples[0];
  t.speed = now - t0 > 300 ? ((t.loaded - l0) / (now - t0)) * 1000 : t.speed;
}

async function run(t) {
  t.status = "running";
  t._started = performance.now();
  t._samples = [[performance.now(), t.base]];
  const chunkMb = app.user?.fileLoading?.uploadChunkSizeMb ?? 10;
  const chunk = chunkMb * MB;
  try {
    if (t.base === 0) await ensureDirs(t);
    if (!chunk || t.size <= chunk) {
      await sendPart(t, await bytesOf(t._file, 0, t.size), 0, t.size, false);
      t.base = t.size;
    } else {
      while (t.base < t.size && t.status === "running") {
        const end = Math.min(t.base + chunk, t.size);
        await sendPart(t, await bytesOf(t._file, t.base, end), t.base, t.size, true);
        t.base = end;
      }
    }
    if (t.status === "running") {
      t.status = "done";
      t.loaded = t.size;
      t.speed = 0;
      t._file = null;
      setTimeout(() => { t.hidden = true; }, 8000);
      fire(t);
    }
  } catch (e) {
    if (e.aborted) {
      if (t.status === "running") t.status = "paused";
    } else {
      t.status = "error";
      t.error = e.message;
      t.speed = 0;
      toastError(new Error(`${t.name}: ${e.message}`));
    }
  } finally {
    t._xhr = null;
    trim();
    pump();
  }
}

export async function pause(t) {
  if (t.status !== "running") return;
  t.status = "paused";
  try { await request("resources/pause", { method: "POST", params: { source: t.source, path: t.path }, raw: true }); } catch (_e) { /* segue */ }
  t._xhr?.abort();
  t.speed = 0;
}

export function resume(t) {
  if (t.status !== "paused") return;
  t.status = "queued";
  t.loaded = t.base;
  pump();
}

export async function cancel(t) {
  const was = t.status;
  t.status = "canceled";
  t._xhr?.abort();
  t._file = null;
  if (was === "running" || was === "paused") {
    try { await filesApi.remove([{ source: t.source, path: t.path }]); } catch (_e) { /* arquivo parcial pode não existir */ }
  }
  pump();
}

export function clearFinished() {
  transfers.items = transfers.items.filter((t) => ["queued", "running", "paused"].includes(t.status));
}

export function dismiss(t) {
  t.hidden = true;
}

export const eta = (t) => (t.speed > 0 ? (t.size - t.loaded) / t.speed : NaN);
export { toast };
