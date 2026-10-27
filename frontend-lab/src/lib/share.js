import * as filesApi from "@/api/files";
import { toastError } from "@/store/app";
import { openShare } from "@/store/shareDlg";

// Cria um link público para um item e mostra o endereço para copiar.
export async function shareItem(source, item) {
  try {
    const existing = await filesApi.shareFor(source, item.path).catch(() => []);
    const link = Array.isArray(existing) && existing.length ? existing[0] : null;
    await openShare(source, item, link);
  } catch (e) { toastError(e); }
}
