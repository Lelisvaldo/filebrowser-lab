import { get, send } from "./client";

// Pendrives: o backend em Go conversa com o usb-manager do servidor.
export async function listUsb() {
  return get("usb", undefined, { quiet401: true });
}

export const ejectUsb = (uuid) => send("usb/" + encodeURIComponent(uuid) + "/eject", "POST");
