// Nuvens (rclone no servidor): login pela interface. Só administradores.
import { send, request } from "./client";

export const list = () => request("lab/cloud");
export const start = (id) => send("lab/cloud/start", "POST", undefined, { id });
export const saveApp = (id, clientId, clientSecret) => send("lab/cloud/app", "PUT", { clientId, clientSecret }, { id });
export const passwordLogin = (id, user, pass) => send("lab/cloud/senha", "POST", { user, pass }, { id });
export const setReadOnly = (id, readOnly) => send("lab/cloud/mode", "POST", undefined, { id, readOnly });
export const disconnect = (id) => send("lab/cloud", "DELETE", undefined, { id });
