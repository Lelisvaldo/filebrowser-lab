// Lixeira: itens apagados das pastas do servidor (30 dias) e a lixeira do Google Drive (só administrador).
import { send, request } from "./client";

export const list = () => request("lab/trash");
export const restore = (items) => send("lab/trash/restore", "POST", { items });
export const purge = (items) => send("lab/trash/purge", "POST", { items });
export const purgeAll = () => send("lab/trash/purge", "POST", { all: true });

export const driveList = () => request("lab/cloud/trash");
export const driveRestore = (ids) => send("lab/cloud/trash/restore", "POST", { ids });
export const drivePurge = (ids) => send("lab/cloud/trash/purge", "POST", { ids });
export const driveEmpty = () => send("lab/cloud/trash/purge", "POST", { all: true });
