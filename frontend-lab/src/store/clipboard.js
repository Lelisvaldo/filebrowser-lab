// Área de transferência interna (Ctrl+C / Ctrl+X / Ctrl+V entre pastas e fontes).
import { reactive } from "vue";
export const clip = reactive({ mode: "", source: "", items: [] });
export function setClip(mode, source, items) { clip.mode = mode; clip.source = source; clip.items = items.map((i) => ({ name: i.name, path: i.path, type: i.type })); }
export function clearClip() { clip.mode = ""; clip.source = ""; clip.items = []; }
