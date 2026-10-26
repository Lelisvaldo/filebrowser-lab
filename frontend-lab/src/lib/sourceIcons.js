// Ícone da pasta raiz pelo nome (padrão: disco).
const BY_SOURCE = { games: "game", media: "film", downloads: "down", private: "lock", pendrives: "usb" };
export const iconForSource = (name) => BY_SOURCE[String(name || "").toLowerCase()] || "hdd";
