// Logo e cor oficial de cada nuvem (logos do simple-icons, domínio público). A chave é o nome da pasta em minúsculas.
export const BRANDS = {
  mega: { key: "mega", name: "Mega", icon: "b-mega", color: "#D9272E" },
  onedrive: { key: "onedrive", name: "OneDrive", icon: "b-onedrive", color: "#0078D4" },
  googledrive: { key: "googledrive", name: "GoogleDrive", icon: "b-drive", color: "#1FA463" },
  dropbox: { key: "dropbox", name: "Dropbox", icon: "b-dropbox", color: "#0061FF" },
};
export const CLOUD_SOURCE = "Nuvens";
// provedor da API (id) -> marca
export const brandById = (id) => BRANDS[id === "drive" ? "googledrive" : id] || null;
// Marca da nuvem a que um caminho da fonte Nuvens pertence (null fora dela).
export function brandOf(source, path) {
  if (source !== CLOUD_SOURCE) return null;
  const first = String(path || "").split("/").filter(Boolean)[0];
  return (first && BRANDS[first.toLowerCase()]) || null;
}
