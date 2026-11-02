// O que acontece com um item apagado, conforme a pasta onde ele está (texto do aviso de apagar).
import { trash } from "@/store/trash";

const CLOUD_SOURCE = "Nuvens";

export function deleteNotice(source, path) {
  if (source === "Pendrives") return { permanent: true, text: "Pendrive não tem lixeira: será apagado direto, sem como recuperar." };
  if (source === CLOUD_SOURCE) {
    const nuvem = String(path || "").split("/").filter(Boolean)[0] || "";
    if (/^GoogleDrive$/i.test(nuvem)) return { permanent: false, text: "Vai para a lixeira do Google Drive; você a vê e restaura na tela Lixeira (administrador)." };
    const nome = nuvem || "da nuvem";
    return { permanent: false, text: `Vai para a lixeira do ${nome} (no site do serviço, que a esvazia sozinho depois de um tempo). Não aparece na lixeira do nosso aplicativo.` };
  }
  return { permanent: false, text: `Vai para a Lixeira por ${trash.days} dias e dá para restaurar. O espaço só é liberado quando ela for esvaziada.` };
}
