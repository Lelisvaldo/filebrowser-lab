// Arrastar itens (dentro do app) para uma pasta: mover. Arquivos vindos de fora seguem o envio normal.
import * as filesApi from "@/api/files";
import { toast, toastError } from "@/store/app";
import { join } from "@/lib/paths";

export const DND = "application/x-lab-items";
export const isLabDrag = (e) => !!e.dataTransfer && Array.from(e.dataTransfer.types || []).includes(DND);

export function startDrag(e, source, list) {
  e.dataTransfer.setData(DND, JSON.stringify({ source, items: list.map((i) => ({ path: i.path, name: i.name, dir: i.type === "directory" })) }));
  e.dataTransfer.effectAllowed = "move";
}

export async function dropInto(e, destSource, destDir) {
  let d;
  try { d = JSON.parse(e.dataTransfer.getData(DND)); } catch (_e) { return; }
  const dest = destDir.endsWith("/") ? destDir : destDir + "/";
  const list = (d.items || []).filter((i) => !(d.source === destSource && (join(dest, i.name, false).replace(/\/$/, "") === i.path.replace(/\/$/, "") || (i.dir && dest.startsWith(i.path.endsWith("/") ? i.path : i.path + "/")))));
  if (!list.length) return;
  try {
    const res = await filesApi.moveCopy(list.map((i) => ({ fromSource: d.source, from: i.path, toSource: destSource, to: join(dest, i.name, i.dir) })), "move");
    const failed = res?.failed?.length || 0;
    toast(failed ? `${failed} item(ns) não puderam ser movidos.` : list.length === 1 ? "Movido." : `${list.length} itens movidos.`, failed ? "err" : "ok");
    window.dispatchEvent(new CustomEvent("lab:refresh"));
  } catch (err) { let m = (err && err.message) || String(err); try { const j = JSON.parse(m); const f = j.failed && j.failed[0] && j.failed[0].message; if (f) m = /itself/i.test(f) ? "O item já está nessa pasta." : f; } catch (_e) { /* texto simples */ } toast(m, "err", 6000); }
}
