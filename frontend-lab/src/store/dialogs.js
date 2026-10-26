// Diálogos próprios (o navegador não mostra alert/prompt no app instalado).
import { reactive } from "vue";

export const dialog = reactive({ open: false, kind: "", title: "", message: "", label: "", value: "", confirm: "OK", danger: false, resolve: null, extra: null });

function open(kind, opts) {
  return new Promise((resolve) => {
    Object.assign(dialog, { open: true, kind, title: "", message: "", label: "", value: "", confirm: "OK", danger: false, extra: null }, opts, { resolve });
  });
}

export const askText = (opts) => open("text", opts);
export const askConfirm = (opts) => open("confirm", opts);
// Escolhe uma pasta de destino: resolve { source, path } ou null.
export const pickFolder = (opts) => open("folder", opts);
export const showInfo = (opts) => open("info", opts);

export function closeDialog(value) {
  const r = dialog.resolve;
  dialog.open = false;
  dialog.resolve = null;
  if (r) r(value);
}
