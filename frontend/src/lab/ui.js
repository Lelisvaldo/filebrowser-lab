// Estado da interface do filebrowser-lab: menu recolhido, densidade e painel de configurações.
// Guardado só neste navegador (localStorage). Tudo opcional: se o storage falhar, usa os padrões.
import { reactive, watch } from "vue";
import { getters, mutations, state } from "@/store";
import { isMobileLayout } from "@/utils/viewport.js";

const KEY = "lab.ui";
const RAIL_EM = 4.5; // largura do menu recolhido (4,5 em ≈ 72 px)

function load() {
  try {
    return JSON.parse(localStorage.getItem(KEY) || "{}");
  } catch (_e) {
    return {};
  }
}

const saved = load();

export const labUi = reactive({
  rail: saved.rail === true,
  density: saved.density === "compact" ? "compact" : "comfortable",
  settingsOpen: false,
});

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify({ rail: labUi.rail, density: labUi.density }));
  } catch (_e) {
    // sem storage: segue com o padrão
  }
}

function userSidebarWidth() {
  const w = parseFloat(localStorage.getItem("sidebarWidth") || "");
  return Number.isFinite(w) && w >= state.sidebar.minWidth ? w : 20;
}

/** No desktop, o menu recolhido vira uma coluna estreita; no celular nada muda. */
function syncWidth() {
  const rail = labUi.rail && !isMobileLayout.value;
  document.documentElement.classList.toggle("lab-rail", rail);
  state.sidebar.width = rail ? RAIL_EM : userSidebarWidth();
}

export function applyLabUi() {
  document.documentElement.classList.toggle("lab-compact", labUi.density === "compact");
  syncWidth();
}

watch(() => [labUi.rail, isMobileLayout.value], syncWidth);

export function setRail(value) {
  labUi.rail = !!value;
  persist();
}

export function toggleRail() {
  setRail(!labUi.rail);
}

export function setDensity(value) {
  labUi.density = value === "compact" ? "compact" : "comfortable";
  persist();
  applyLabUi();
}

/** Botão de menu da barra: recolhe/expande no desktop; no celular abre e fecha o menu. */
export function labToggleMenu() {
  if (getters.isMobile()) {
    mutations.toggleSidebar();
  } else {
    toggleRail();
  }
}

export function openSettingsPanel() {
  labUi.settingsOpen = true;
}

export function closeSettingsPanel() {
  labUi.settingsOpen = false;
}
