import * as filesApi from "@/api/files";
import { askConfirm, showInfo } from "@/store/dialogs";
import { toast, toastError } from "@/store/app";

// Cria um link público para um item e mostra o endereço para copiar.
export async function shareItem(source, item) {
  try {
    const existing = await filesApi.shareFor(source, item.path).catch(() => []);
    let link = Array.isArray(existing) && existing.length ? existing[0] : null;
    if (!link) {
      const ok = await askConfirm({
        title: "Compartilhar com link",
        message: `Qualquer pessoa com o link poderá baixar "${item.name}". O link vale por 7 dias.`,
        confirm: "Criar link",
      });
      if (!ok) return;
      link = await filesApi.createShare({ path: item.path, source, expires: "7", unit: "days" });
    }
    const url = link.shareURL || `${location.origin}/public/share/${link.hash}`;
    try { await navigator.clipboard.writeText(url); toast("Link copiado."); } catch (_e) { /* mostra no diálogo */ }
    await showInfo({ title: "Link de compartilhamento", message: url, confirm: "Fechar" });
  } catch (e) { toastError(e); }
}
