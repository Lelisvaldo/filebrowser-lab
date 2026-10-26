// Variáveis que o backend injeta em <script id="app-config">.
function read() {
  try {
    const el = document.getElementById("app-config");
    if (el && el.textContent) return JSON.parse(el.textContent);
  } catch (_e) { /* cai no padrão */ }
  return {};
}

export const cfg = read();
export const baseURL = (cfg.baseURL || "/").replace(/\/?$/, "/");
export const appName = cfg.name || "filebrowser-lab";
