import { reactive } from "vue";
import * as authApi from "@/api/auth";
import * as filesApi from "@/api/files";
import { listUsb, hostName } from "@/api/usb";
import { setUnauthorizedHandler } from "@/api/client";

const KEY = "lab.ui";
const USB_SOURCE = "Pendrives";

function load() {
  try { return JSON.parse(localStorage.getItem(KEY) || "{}"); } catch (_e) { return {}; }
}
const saved = load();

// Preferências só deste navegador
export const ui = reactive({
  theme: ["light", "dark"].includes(saved.theme) ? saved.theme : "auto",
  density: saved.density === "compact" ? "compact" : "comfortable",
  rail: saved.rail === true,
  view: saved.view === "grid" ? "grid" : "list",
  sortBy: ["name", "size", "modified"].includes(saved.sortBy) ? saved.sortBy : "name",
  sortAsc: saved.sortAsc !== false,
  usbRefresh: [3, 10, 0].includes(saved.usbRefresh) ? saved.usbRefresh : 5,
  tcomp: saved.tcomp === true,
  settingsOpen: false,
  menuOpen: false,
  transfersOpen: false,
  usbOpen: false,
});

export const app = reactive({
  ready: false,
  user: null,
  sources: {},     // nome -> info de uso (settings/sources)
  host: "",        // nome do servidor (hostname)
  usb: null,       // lista do usb-manager, ou null se a integração não existe
  toasts: [],
});

export function persist() {
  try {
    const { theme, density, rail, view, sortBy, sortAsc, usbRefresh, tcomp } = ui;
    localStorage.setItem(KEY, JSON.stringify({ theme, density, rail, view, sortBy, sortAsc, usbRefresh, tcomp }));
  } catch (_e) { /* sem storage: segue */ }
}

const mqMobile = window.matchMedia("(max-width:760px)");
export const isMobile = () => mqMobile.matches;

export function applyUi() {
  const root = document.documentElement;
  if (ui.theme === "auto") root.removeAttribute("data-theme");
  else root.setAttribute("data-theme", ui.theme);
  document.body.classList.toggle("compact", ui.density === "compact");
  document.body.classList.toggle("rail", ui.rail && !isMobile());
  document.body.classList.toggle("menu", ui.menuOpen && isMobile());
  document.body.classList.toggle("cfg-open", ui.settingsOpen);
}

export function setUi(patch) {
  Object.assign(ui, patch);
  persist();
  applyUi();
}

export const isDark = () =>
  ui.theme === "dark" || (ui.theme === "auto" && window.matchMedia("(prefers-color-scheme:dark)").matches);

export function toast(text, kind = "ok", ms = 4200) {
  const t = { id: Math.random().toString(36).slice(2), text, kind };
  app.toasts.push(t);
  setTimeout(() => { app.toasts = app.toasts.filter((x) => x.id !== t.id); }, ms);
}

export const toastError = (e) => toast((e && e.message) || String(e), "err", 6000);

// ---- sessão ----
let renewTimer = null;
export async function loadSession() {
  try {
    app.user = await authApi.me();
  } catch (_e) {
    app.user = null;
  }
  app.ready = true;
  if (app.user) {
    startRenew();
    await refreshSources();
    void refreshUsb();
    hostName().then((h) => { app.host = (h && h.name) || ""; }).catch(() => {});
  }
  return app.user;
}

function startRenew() {
  clearInterval(renewTimer);
  renewTimer = setInterval(() => { authApi.renew().catch(() => {}); }, 10 * 60 * 1000);
}

export async function doLogout() {
  clearInterval(renewTimer);
  await authApi.logout();
  app.user = null;
  app.sources = {};
  app.usb = null;
}

export async function refreshSources() {
  try { app.sources = await filesApi.sources(); } catch (_e) { /* mantém o anterior */ }
}

// ---- fontes visíveis ao usuário ----
export const hasUsbSource = () => (app.user?.scopes || []).some((s) => s.name === USB_SOURCE);
export const userSources = () => (app.user?.scopes || []).filter((s) => s.name !== USB_SOURCE);
export const isAdmin = () => !!app.user?.permissions?.admin;
export const can = (perm) => !!app.user?.permissions?.[perm];
export function scopeFor(source) {
  return (app.user?.scopes || []).find((s) => s.name === source);
}
export const canIn = (source, perm) => !!scopeFor(source)?.permissions?.[perm];
export const usbSourceName = USB_SOURCE;

// ---- pendrives ----
let usbTimer = null;
let usbAbsent = false;
export async function refreshUsb() {
  if (usbAbsent) return;
  try {
    const list = await listUsb();
    app.usb = Array.isArray(list) ? list : null;
  } catch (e) {
    if (e.status === 404 || e.status === 501) { app.usb = null; usbAbsent = true; } // sem integração: para de perguntar
  }
}

export function startUsbPolling() {
  clearInterval(usbTimer);
  if (ui.usbRefresh > 0) usbTimer = setInterval(() => { if (app.user && !document.hidden && !usbAbsent) void refreshUsb(); }, ui.usbRefresh * 1000);
}

setUnauthorizedHandler(() => {
  if (app.user) {
    app.user = null;
    window.dispatchEvent(new Event("lab:session-expired"));
  }
});

mqMobile.addEventListener("change", applyUi);

// ---- título da página (barra superior) ----
export const page = reactive({ title: "Este servidor" });

// ---- ejeção segura ----
import { ejectUsb } from "@/api/usb";
export const ejecting = reactive({});

export async function eject(dev) {
  if (ejecting[dev.uuid]) return;
  ejecting[dev.uuid] = true;
  try {
    await ejectUsb(dev.uuid);
    toast(`${dev.name}: desligado com segurança. Pode tirar o pendrive.`);
  } catch (e) {
    const busy = e.status === 409 || /busy|em uso/i.test(e.message || "");
    toast(busy ? `${dev.name} ainda está em uso. Feche os arquivos abertos e tente de novo.` : `Não foi possível remover ${dev.name}: ${e.message}`, "err", 7000);
  } finally {
    delete ejecting[dev.uuid];
    void refreshUsb();
  }
}

// ---- rodapé (barra de status fixa no fim da página) ----
export const status = reactive({ show: false, label: "", used: 0, total: 0, count: 0, folders: 0, files: 0, size: 0, selCount: 0, selSize: 0, noun: "itens" });
export const hideStatus = () => { status.show = false; };
