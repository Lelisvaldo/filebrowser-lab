// Quais pastas estão abertas na árvore do menu lateral (lembrado neste navegador).
import { reactive } from "vue";
const KEY = "lab.tree2"; // pastas começam fechadas; só abrem quando o usuário abre
let saved = [];
try { saved = JSON.parse(localStorage.getItem(KEY) || "[]"); } catch (_e) { /* sem storage */ }
const open = reactive(new Set(Array.isArray(saved) ? saved : []));
const k = (source, path) => source + ":" + ("/" + String(path).split("/").filter(Boolean).join("/"));
const save = () => { try { localStorage.setItem(KEY, JSON.stringify([...open])); } catch (_e) { /* sem storage */ } };
export const isOpen = (source, path) => open.has(k(source, path));
export function toggleOpen(source, path) { const key = k(source, path); if (open.has(key)) open.delete(key); else open.add(key); save(); }
/** Abre os pais da pasta atual para ela aparecer na árvore. */
export function reveal(source, dirPath) {
  const parts = String(dirPath).split("/").filter(Boolean);
  let acc = "/";
  open.add(k(source, "/"));
  for (const p of parts.slice(0, -1)) { acc += p + "/"; open.add(k(source, acc)); }
  save();
}
