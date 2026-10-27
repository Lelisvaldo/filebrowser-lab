// Personalização de pastas (ícone e cor) e atalhos fixados no menu lateral. Fica só neste navegador.
import { reactive } from "vue";
import { hueOf } from "./sourceIcons";

const KEY = "lab.folders";
let saved = {};
try { saved = JSON.parse(localStorage.getItem(KEY) || "{}") || {}; } catch (_e) { /* sem storage */ }
export const prefs = reactive({ styles: saved.styles || {}, pins: Array.isArray(saved.pins) ? saved.pins : [] });
const save = () => { try { localStorage.setItem(KEY, JSON.stringify({ styles: prefs.styles, pins: prefs.pins })); } catch (_e) { /* sem storage */ } };

export const STYLE_ICONS = ["folder", "star", "game", "film", "music", "image", "video", "lock", "home", "hdd", "code", "people", "down", "usb", "text", "table", "zip", "shield", "key", "gear"];
export const STYLE_HUES = ["cyan", "violet", "magenta", "pink", "orange", "green", "blue", "yellow", "red"];

export const keyOf = (source, path) => source + ":" + ("/" + String(path).split("/").filter(Boolean).join("/"));

export function setStyle(source, path, st) {
  const k = keyOf(source, path);
  if (!st || (!st.icon && !st.hue)) delete prefs.styles[k];
  else prefs.styles[k] = { icon: st.icon || "", hue: st.hue || "" };
  save();
}

/** Ícone e classe de cor de uma pasta: o que o usuário escolheu, senão a cor da pasta raiz. */
export function folderLook(source, path) {
  const s = prefs.styles[keyOf(source, path)];
  return { icon: (s && s.icon) || "folder", cls: s && s.hue ? "hue h-" + s.hue : hueOf(source), custom: !!s };
}

export const isPinned = (source, path) => prefs.pins.some((p) => p.source === source && keyOf(p.source, p.path) === keyOf(source, path));
export function togglePin(source, path, name) {
  if (isPinned(source, path)) prefs.pins = prefs.pins.filter((p) => keyOf(p.source, p.path) !== keyOf(source, path));
  else prefs.pins = [...prefs.pins, { source, path: "/" + String(path).split("/").filter(Boolean).join("/") + (path === "/" ? "" : "/"), name: name || String(path).split("/").filter(Boolean).pop() || source }];
  save();
}
