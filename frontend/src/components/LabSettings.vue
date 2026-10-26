<!-- eslint-disable @intlify/vue-i18n/no-raw-text, vue/no-v-html -->
<template>
  <div v-if="labUi.settingsOpen" class="lab-set-root">
    <div class="lab-set-scrim" @click="close"></div>
    <aside class="lab-set" role="dialog" aria-modal="true" aria-label="Configurações" @keydown.esc="close">
      <header class="lab-set-head">
        <b>Configurações</b>
        <button ref="closeBtn" type="button" class="lab-set-x" aria-label="Fechar" @click="close">
          <svg viewBox="0 0 24 24" aria-hidden="true" v-html="icons.x"></svg>
        </button>
      </header>

      <div class="lab-set-body">
        <div class="lab-set-account">
          <span class="lab-set-avatar"><svg viewBox="0 0 24 24" aria-hidden="true" v-html="icons.person"></svg></span>
          <div class="lab-set-who">
            <b>{{ username }}</b>
            <span>{{ isAdmin ? "Administrador" : "Usuário" }}</span>
          </div>
          <button v-if="canLogout" type="button" class="lab-set-btn" @click="logout">
            <svg viewBox="0 0 24 24" aria-hidden="true" v-html="icons.signout"></svg><span>Sair</span>
          </button>
        </div>

        <h3>Modo de cor</h3>
        <div class="lab-set-themes" role="group" aria-label="Modo de cor">
          <button type="button" class="lab-set-theme" :aria-pressed="!dark" @click="setDark(false)">
            <span class="lab-sw"><i style="background:#e6e9ed"></i><i style="background:#0b62b8"></i></span>Claro
          </button>
          <button type="button" class="lab-set-theme" :aria-pressed="dark" @click="setDark(true)">
            <span class="lab-sw"><i style="background:#1f0c4c"></i><i style="background:#01ced3"></i></span>Escuro
          </button>
        </div>
        <p class="lab-set-hint">A escolha fica salva na sua conta.</p>

        <h3>Aparência</h3>
        <div class="lab-set-opt">
          <div class="lab-set-l"><b>Densidade</b><span>Linhas mais baixas mostram mais itens.</span></div>
          <div class="lab-set-seg" role="group" aria-label="Densidade">
            <button type="button" :aria-pressed="labUi.density === 'comfortable'" @click="setDensity('comfortable')">Confortável</button>
            <button type="button" :aria-pressed="labUi.density === 'compact'" @click="setDensity('compact')">Compacta</button>
          </div>
        </div>
        <div class="lab-set-opt">
          <div class="lab-set-l"><b>Menu lateral recolhido</b><span>Só ícones, para ganhar espaço.</span></div>
          <button type="button" class="lab-set-sw" role="switch" aria-label="Menu lateral recolhido"
            :aria-checked="labUi.rail" @click="setRail(!labUi.rail)"></button>
        </div>

        <h3>Mais</h3>
        <a v-if="settingsAllowed" class="lab-set-link" href="/settings" @click.prevent="goAdvanced">
          Configurações avançadas<small>Usuários, compartilhamentos, fontes e perfil</small>
        </a>
      </div>
    </aside>
  </div>
</template>

<script>
import { getters, mutations, state } from "@/store";
import { logout } from "@/utils/auth";
import { ICONS } from "@/lab/icons";
import { labUi, closeSettingsPanel, setDensity, setRail } from "@/lab/ui";

export default {
  name: "LabSettings",
  data() {
    return { labUi, icons: ICONS };
  },
  computed: {
    dark() {
      return getters.isDarkMode();
    },
    username() {
      return state.user?.username || "";
    },
    isAdmin() {
      return getters.isAdmin();
    },
    canLogout() {
      return getters.isLoggedIn() && state.user?.username !== "anonymous";
    },
    settingsAllowed() {
      return getters.isLoggedIn() && !state.user?.disableSettings;
    },
  },
  watch: {
    "labUi.settingsOpen"(open) {
      if (open) {
        this.$nextTick(() => this.$refs.closeBtn && this.$refs.closeBtn.focus());
      }
    },
  },
  mounted() {
    document.addEventListener("keydown", this.onKey);
  },
  beforeUnmount() {
    document.removeEventListener("keydown", this.onKey);
  },
  methods: {
    setDensity,
    setRail,
    close: closeSettingsPanel,
    onKey(e) {
      if (e.key === "Escape" && labUi.settingsOpen) {
        closeSettingsPanel();
      }
    },
    setDark(wanted) {
      if (wanted !== getters.isDarkMode()) {
        mutations.toggleDarkMode();
      }
    },
    goAdvanced() {
      closeSettingsPanel();
      void this.$router.push("/settings");
    },
    async logout() {
      closeSettingsPanel();
      await logout();
    },
  },
};
</script>

<style scoped>
.lab-set-root { position: fixed; inset: 0; z-index: 200; }
.lab-set-scrim { position: absolute; inset: 0; background: rgba(0, 0, 0, 0.45); }
.lab-set { position: absolute; top: 0; right: 0; bottom: 0; width: min(380px, 100%); display: flex; flex-direction: column; background: var(--background); color: var(--textPrimary); border-left: 1px solid var(--divider); box-shadow: -10px 0 40px rgba(0, 0, 0, 0.35); padding-top: var(--safe-area-top); }
.lab-set-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; min-height: 56px; padding: 0 0.5rem 0 1rem; border-bottom: 1px solid var(--divider); font-size: 1.05rem; }
.lab-set-x { display: grid; place-items: center; width: 40px; height: 40px; padding: 0; border: 0; border-radius: 10px; background: transparent; color: var(--textSecondary); cursor: pointer; }
.lab-set-x:hover { background: color-mix(in srgb, var(--primaryColor) 12%, transparent); color: var(--textPrimary); }
.lab-set-x svg { width: 20px; height: 20px; fill: currentColor; }
.lab-set-body { flex: 1; overflow-y: auto; padding: 0.25rem 1rem 2rem; padding-right: calc(1rem + var(--safe-area-right)); }
h3 { margin: 1.4rem 0 0.4rem; font-size: 0.72rem; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--textSecondary); }
.lab-set-account { display: flex; align-items: center; gap: 0.75rem; padding: 0.9rem 0; border-bottom: 1px solid var(--divider); }
.lab-set-avatar { display: grid; place-items: center; width: 40px; height: 40px; border-radius: 99px; background: color-mix(in srgb, var(--primaryColor) 18%, transparent); color: var(--primaryColor); }
.lab-set-avatar svg { width: 22px; height: 22px; fill: currentColor; }
.lab-set-who { flex: 1; min-width: 0; line-height: 1.25; }
.lab-set-who b { display: block; font-weight: 600; overflow-wrap: anywhere; }
.lab-set-who span { font-size: 0.85rem; color: var(--textSecondary); }
.lab-set-btn { display: inline-flex; align-items: center; gap: 0.4rem; min-height: 40px; padding: 0 0.8rem; border: 1px solid var(--divider); border-radius: 10px; background: var(--surfacePrimary); color: var(--textPrimary); cursor: pointer; font: inherit; font-weight: 600; }
.lab-set-btn:hover { background: color-mix(in srgb, var(--primaryColor) 12%, var(--surfacePrimary)); }
.lab-set-btn svg { width: 18px; height: 18px; fill: currentColor; }
.lab-set-themes { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0.5rem; padding-top: 0.5rem; }
.lab-set-theme { display: flex; flex-direction: column; gap: 0.4rem; padding: 0.5rem; border: 2px solid var(--divider); border-radius: 12px; background: var(--surfacePrimary); color: var(--textPrimary); cursor: pointer; font: inherit; font-weight: 600; }
.lab-set-theme[aria-pressed="true"] { border-color: var(--primaryColor); color: var(--primaryColor); }
.lab-sw { display: flex; height: 44px; overflow: hidden; border: 1px solid rgba(128, 128, 128, 0.35); border-radius: 8px; }
.lab-sw i { flex: 1; }
.lab-set-hint { margin: 0.5rem 0 0; font-size: 0.82rem; color: var(--textSecondary); }
.lab-set-opt { display: flex; align-items: center; gap: 0.75rem; padding: 0.85rem 0; border-bottom: 1px solid var(--divider); }
.lab-set-l { flex: 1; min-width: 0; line-height: 1.3; }
.lab-set-l b { display: block; font-weight: 600; }
.lab-set-l span { font-size: 0.85rem; color: var(--textSecondary); }
.lab-set-seg { display: flex; flex: none; overflow: hidden; border: 1px solid var(--divider); border-radius: 10px; }
.lab-set-seg button { min-height: 36px; padding: 0 0.75rem; border: 0; background: var(--surfacePrimary); color: var(--textSecondary); cursor: pointer; font: inherit; }
.lab-set-seg button[aria-pressed="true"] { background: color-mix(in srgb, var(--primaryColor) 20%, transparent); color: var(--primaryColor); font-weight: 600; }
.lab-set-sw { position: relative; flex: none; width: 46px; height: 26px; min-width: 46px; min-height: 26px; padding: 0; border: 0; border-radius: 99px; background: var(--surfaceSecondary); cursor: pointer; }
.lab-set-sw::after { content: ""; position: absolute; top: 3px; left: 3px; width: 20px; height: 20px; border-radius: 99px; background: #fff; transition: transform 0.15s; }
.lab-set-sw[aria-checked="true"] { background: var(--primaryColor); }
.lab-set-sw[aria-checked="true"]::after { transform: translateX(20px); }
.lab-set-link { display: block; margin-top: 0.4rem; padding: 0.8rem 0.9rem; border: 1px solid var(--divider); border-radius: 12px; background: var(--surfacePrimary); color: var(--textPrimary); text-decoration: none; font-weight: 600; }
.lab-set-link small { display: block; font-weight: 400; color: var(--textSecondary); }
.lab-set-link:hover { border-color: var(--primaryColor); }
:is(button, a):focus-visible { outline: 2px solid var(--primaryColor); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { .lab-set-sw::after { transition: none; } }
</style>
