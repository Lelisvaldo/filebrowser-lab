<!-- eslint-disable @intlify/vue-i18n/no-raw-text, vue/no-v-html -->
<template>
  <div class="lab-overview">
    <svg class="lab-defs" width="0" height="0" aria-hidden="true">
      <defs>
        <linearGradient id="lab-ov-a" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#0bc7d3" /><stop offset=".55" stop-color="#7a76d4" /><stop offset="1" stop-color="#f31ed2" /></linearGradient>
        <linearGradient id="lab-ov-b" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#00c6cf" /><stop offset="1" stop-color="#065e9e" /></linearGradient>
        <linearGradient id="lab-ov-c" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ff9630" /><stop offset="1" stop-color="#ff1bca" /></linearGradient>
      </defs>
    </svg>

    <header class="lab-head">
      <h1>Este servidor</h1>
      <nav class="lab-crumbs" aria-label="Caminho"><span>Início</span></nav>
    </header>
    <p class="lab-sub">Toque numa pasta para ver os arquivos.</p>

    <section v-if="sources.length" aria-labelledby="lab-h-pastas">
      <h2 id="lab-h-pastas">Pastas <small>· {{ sources.length }}</small></h2>
      <div class="lab-tiles">
        <a v-for="s in sources" :key="s.name" class="lab-tile" :href="hrefFor(s)" :aria-label="s.name"
          @click.prevent="open(s)">
          <div class="lab-row">
            <span class="lab-ico"><svg viewBox="0 0 24 24" aria-hidden="true" v-html="iconFor(s.name)"></svg></span>
            <div class="lab-name">
              <div class="lab-nm">{{ s.name }}</div>
              <div class="lab-meta">{{ statusText(s) }}</div>
            </div>
            <svg class="lab-donut" viewBox="0 0 100 100" role="img" :aria-label="s.pct + '% usado'">
              <circle cx="50" cy="50" r="46" fill="none" :stroke="ringFor(s.i)" stroke-width="3" />
              <circle cx="50" cy="50" r="34" fill="none" class="lab-track" stroke-width="15" />
              <circle cx="50" cy="50" r="34" fill="none" :stroke="'url(#' + gradFor(s.i) + ')'" stroke-width="15"
                :stroke-dasharray="(s.pct / 100 * 213.63).toFixed(1) + ' 213.63'" transform="rotate(-90 50 50)" />
              <circle cx="50" cy="50" r="24" class="lab-inner" />
              <text x="50" y="58" text-anchor="middle" font-size="25" font-weight="600" class="lab-pct">{{ s.pct }}%</text>
            </svg>
          </div>
          <div class="lab-foot">
            <span v-if="s.total > 0"><b>{{ fmt(s.used) }}</b> usados</span>
            <span v-else>calculando…</span>
            <span v-if="s.total > 0">{{ fmt(s.total) }}</span>
          </div>
        </a>
      </div>
    </section>

    <p v-else class="lab-empty">Nenhuma pasta disponível para este usuário.</p>
  </div>
</template>

<script>
import { state } from "@/store";
import { globalVars } from "@/utils/constants";
import { goToItem } from "@/utils/url";
import { getHumanReadableFilesize } from "@/utils/filesizes";

// Ícones Fluent (Microsoft, MIT), 24x24.
const ICONS = {"folder": "<path d=\"M3.5 6.25V8h4.63q.31 0 .53-.22l1.53-1.53-1.53-1.53a.8.8 0 0 0-.53-.22H5.25c-.97 0-1.75.78-1.75 1.75m-1.5 0C2 4.45 3.46 3 5.25 3h2.88c.6 0 1.17.24 1.59.66l1.84 1.84h7.19c1.8 0 3.25 1.46 3.25 3.25v9c0 1.8-1.46 3.25-3.25 3.25H5.25A3.25 3.25 0 0 1 2 17.75zM3.5 9.5v8.25c0 .97.78 1.75 1.75 1.75h13.5c.97 0 1.75-.78 1.75-1.75v-9c0-.97-.78-1.75-1.75-1.75h-7.19L9.72 8.84c-.42.42-1 .66-1.6.66z\"/>", "game": "<path d=\"M9.8 5.5a2.6 2.6 0 0 0-1.63-.2l-1.18.23c-.86.16-1.6.72-2 1.5-1.37 2.67-2.4 4.86-2.8 6.73-.42 1.92-.17 3.57 1.07 5.07.82 1 2.23.8 2.97-.03.56-.64 1.2-1.37 1.81-2.09.41-.47 1-.75 1.63-.75h4.66c.62 0 1.22.28 1.63.75l1.82 2.09c.73.83 2.14 1.03 2.96.03 1.24-1.5 1.49-3.15 1.08-5.07-.4-1.87-1.43-4.06-2.8-6.73a2.9 2.9 0 0 0-2-1.5l-1.19-.23c-.53-.1-1.1-.06-1.64.2l-.43.24q-.54.3-1.12.3h-1.28q-.58 0-1.11-.3zM8.45 6.7q.42-.08.73.09l.4.2q.82.46 1.8.47h1.27q.97-.01 1.8-.47l.4-.2q.3-.16.73-.08l1.18.22c.43.08.8.36 1 .75 1.38 2.7 2.32 4.72 2.67 6.38.34 1.59.12 2.77-.78 3.86a.4.4 0 0 1-.35.15.6.6 0 0 1-.44-.22c-.56-.63-1.19-1.35-1.8-2.07a3.6 3.6 0 0 0-2.72-1.25H9.67c-1.04 0-2.03.46-2.7 1.25l-1.81 2.07a.6.6 0 0 1-.45.22.4.4 0 0 1-.35-.15c-.9-1.1-1.12-2.27-.78-3.87.35-1.65 1.3-3.67 2.68-6.37.2-.39.56-.67 1-.75zM12 11a1 1 0 1 0 0-2 1 1 0 0 0 0 2\"/>", "film": "<path d=\"m19.73 3.92.05.16.55 1.92c.1.36-.08.74-.42.9l-.1.03L9.1 10h11.16c.38 0 .7.29.74.65v8.6A2.75 2.75 0 0 1 18.43 22H5.75a2.75 2.75 0 0 1-2.74-2.58L3 19.25v-8.4l-.52-1.81c-.4-1.4.36-2.87 1.72-3.35l.16-.05 12.02-3.45c1.4-.4 2.87.37 3.35 1.73m-.23 7.58h-15v7.75c0 .6.43 1.1 1 1.23l.12.01.13.01h12.5c.65 0 1.18-.5 1.24-1.12V11.5M6.27 6.65l-1.5.43c-.61.18-1 .8-.88 1.42l.03.12.34 1.2.3-.08zm4.76-1.36-2.72.77-1.71 3.1 2.72-.78zm4.76-1.37-2.72.78-1.71 3.1L14.07 7zm1.84-.23-1.52 2.74 2.57-.74-.34-1.2c-.1-.37-.38-.66-.7-.8\"/>", "down": "<path d=\"M18.25 20.5a.75.75 0 1 1 0 1.5h-13a.75.75 0 1 1 0-1.5zm-6.6-18.49h.1c.38 0 .7.28.74.64l.01.1v13.7l3.72-3.73a.75.75 0 0 1 .98-.07l.08.07c.27.27.3.68.07.98l-.07.08-5 5a.75.75 0 0 1-.97.07l-.09-.07-5-5a.75.75 0 0 1 .98-1.13l.08.07L11 16.43V2.76c0-.38.28-.7.65-.75h.1z\"/>", "lock": "<path d=\"M12 1a5 5 0 0 1 5 5v2.01c1.68.13 3 1.53 3 3.24v7.5c0 1.8-1.46 3.25-3.25 3.25h-9.5A3.25 3.25 0 0 1 4 18.75v-7.5a3.25 3.25 0 0 1 3-3.24V6a5 5 0 0 1 5-5M7.25 9.5c-.97 0-1.75.78-1.75 1.75v7.5c0 .97.78 1.75 1.75 1.75h9.5c.97 0 1.75-.78 1.75-1.75v-7.5c0-.97-.78-1.75-1.75-1.75zM12 13.75a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5M12 2.5A3.5 3.5 0 0 0 8.5 6v2h7V6A3.5 3.5 0 0 0 12 2.5\"/>", "hdd": "<path d=\"M5.42 5.27A2.3 2.3 0 0 1 7.44 4h9.12c.86 0 1.65.5 2.02 1.27l3.1 6.39q.31.67.32 1.41v3.68c0 1.24-1 2.25-2.25 2.25H4.25C3.01 19 2 18 2 16.75v-3.68q0-.74.32-1.41zm11.81.65a.8.8 0 0 0-.67-.42H7.44a.8.8 0 0 0-.67.42L4.3 11h15.38zM3.5 13.25v3.5c0 .41.34.75.75.75h15.5c.41 0 .75-.34.75-.75v-3.5a.75.75 0 0 0-.75-.75H4.25a.75.75 0 0 0-.75.75M18 16a1 1 0 1 0 0-2 1 1 0 0 0 0 2\"/>"};
const BY_NAME = { games: "game", media: "film", downloads: "down", private: "lock" };
const GRADS = ["lab-ov-a", "lab-ov-b", "lab-ov-c", "lab-ov-b"];
const RINGS = ["#14d4dc", "#00c6cf", "#ff15d2", "#00c6cf"];

export default {
  name: "overview",
  computed: {
    sources() {
      const scopes = state.user?.scopes || [];
      return scopes.map((sc, i) => {
        const info = state.sources.info?.[sc.name] || {};
        const total = Number(info.total) || 0;
        const used = Number(info.used) || 0;
        const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
        return { name: sc.name, i, used, total, pct, status: info.status || "unknown", prefix: info.pathPrefix || "" };
      });
    },
  },
  methods: {
    fmt(bytes) {
      return getHumanReadableFilesize(bytes);
    },
    iconFor(name) {
      return ICONS[BY_NAME[String(name).toLowerCase()] || "hdd"] || ICONS.folder;
    },
    gradFor(i) {
      return GRADS[i % GRADS.length];
    },
    ringFor(i) {
      return RINGS[i % RINGS.length];
    },
    statusText(s) {
      if (s.status === "ready") return "Indexada";
      if (s.status === "indexing") return "Indexando…";
      return "Pasta";
    },
    hrefFor(s) {
      return `${globalVars.baseURL}files/${s.prefix ? s.prefix + "/" : ""}`;
    },
    open(s) {
      goToItem(s.name, "/", {}, false, false);
    },
  },
};
</script>

<style scoped>
.lab-defs { position: absolute; }
.lab-overview { max-width: 960px; margin: 0 auto; padding: 1.25rem 1rem 3rem; color: var(--textPrimary); }
.lab-head { display: flex; flex-wrap: wrap; align-items: baseline; justify-content: space-between; gap: 0.25rem 1rem; }
.lab-head h1 { margin: 0; font-size: 1.6rem; font-weight: 500; }
.lab-crumbs { font-size: 0.88rem; color: var(--textSecondary); }
.lab-sub { margin: 0.25rem 0 1.25rem; color: var(--textSecondary); }
h2 { display: flex; align-items: center; gap: 0.5rem; margin: 1.5rem 0 0.75rem; font-size: 0.95rem; font-weight: 700; }
h2 small { font-weight: 400; color: var(--textSecondary); }
.lab-tiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 210px), 1fr)); gap: 0.75rem; }
.lab-tile { display: flex; flex-direction: column; gap: 0.5rem; min-width: 0; padding: 0.7rem 0.8rem; text-decoration: none; color: inherit; background: var(--surfacePrimary); border: 1px solid var(--divider); border-radius: 14px; transition: border-color 0.12s, background 0.12s; }
.lab-tile:hover { border-color: var(--primaryColor); }
.lab-tile:focus-visible { outline: 2px solid var(--primaryColor); outline-offset: 2px; }
.lab-row { display: flex; align-items: center; gap: 0.65rem; min-width: 0; }
.lab-ico { display: grid; place-items: center; flex: none; width: 34px; height: 34px; border-radius: 8px; background: color-mix(in srgb, var(--primaryColor) 18%, transparent); color: var(--primaryColor); }
.lab-ico svg { width: 19px; height: 19px; fill: currentColor; }
.lab-name { flex: 1; min-width: 0; line-height: 1.2; }
.lab-nm { font-weight: 600; overflow-wrap: anywhere; }
.lab-meta { font-size: 0.72rem; color: var(--textSecondary); }
.lab-donut { flex: none; width: 52px; height: 52px; }
.lab-track { stroke: var(--surfaceSecondary); }
.lab-inner { fill: var(--iconBackground); }
.lab-pct { fill: var(--textPrimary); }
.lab-foot { display: flex; justify-content: space-between; align-items: baseline; gap: 0.5rem; font-size: 0.78rem; color: var(--textSecondary); }
.lab-foot b { color: var(--textPrimary); font-weight: 600; }
.lab-empty { color: var(--textSecondary); }
</style>
