// Janela de compartilhamento (criar e editar links). openShare() devolve o link salvo, ou null se cancelou/removeu.
import { reactive } from "vue";
export const shareDlg = reactive({ open: false, source: "", item: null, link: null, resolve: null });
export function openShare(source, item, link = null) {
  return new Promise((resolve) => { Object.assign(shareDlg, { open: true, source, item, link, resolve }); });
}
export function closeShare(result = null) {
  const r = shareDlg.resolve;
  Object.assign(shareDlg, { open: false, resolve: null });
  if (r) r(result);
}
