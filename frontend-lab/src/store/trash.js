// Quantos itens há na Lixeira (para o número ao lado do ícone no menu) e o prazo em dias.
import { reactive } from "vue";
import * as trashApi from "@/api/trash";

export const trash = reactive({ count: 0, days: 30 });

export async function refreshTrash() {
  try {
    const r = await trashApi.list();
    trash.count = r?.items?.length || 0;
    trash.days = r?.days || 30;
  } catch (_e) { /* sem lixeira ou sem rede: o menu só não mostra o número */ }
}
