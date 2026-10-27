// Ícone da pasta raiz pelo nome (padrão: disco).
const BY_SOURCE = { games: "game", media: "film", downloads: "down", private: "lock", pendrives: "usb", nuvens: "cloud" };
export const iconForSource = (name) => BY_SOURCE[String(name || "").toLowerCase()] || "hdd";

// Cor de cada pasta raiz (classe h-*): as conhecidas têm cor fixa, as outras giram pela paleta.
const HUES = ["cyan", "violet", "magenta", "orange", "green", "blue", "yellow", "red"];
const HUE_BY_SOURCE = { games: "violet", media: "magenta", downloads: "cyan", private: "orange", pendrives: "green", nuvens: "cyan" };
export function hueOf(name) {
  const k = String(name || "").toLowerCase();
  if (HUE_BY_SOURCE[k]) return "hue h-" + HUE_BY_SOURCE[k];
  let n = 0;
  for (const ch of k) n = (n * 31 + ch.charCodeAt(0)) >>> 0;
  return "hue h-" + HUES[n % HUES.length];
}
