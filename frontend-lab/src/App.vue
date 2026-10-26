<script setup>
import { computed, watch, onMounted, onBeforeUnmount, ref, nextTick } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import DialogHost from "@/components/DialogHost.vue";
import Toasts from "@/components/Toasts.vue";
import SettingsDrawer from "@/components/SettingsDrawer.vue";
import TransfersPop from "@/components/TransfersPop.vue";
import TransferDock from "@/components/TransferDock.vue";
import { app, ui, status, setUi, page, userSources, hasUsbSource, isMobile, isDark, eject, ejecting, startUsbPolling, refreshUsb, refreshSources } from "@/store/app";
import { visible, active, onTransferDone } from "@/store/transfers";
import { iconForSource } from "@/lib/sourceIcons";
import { routeFor } from "@/lib/paths";
import { appName } from "@/lib/config";
import { bytes } from "@/lib/format";

const route = useRoute();
const router = useRouter();
const q = ref("");
const searchOpen = ref(false);

const logged = computed(() => !!app.user && !route.meta.public);
const activeCount = computed(() => visible().filter((t) => ["queued", "running", "paused"].includes(t.status)).length);
const usbCount = computed(() => (app.usb || []).length);
const usbOn = computed(() => app.usb !== null);

function toggleMenu() {
  if (isMobile()) setUi({ menuOpen: !ui.menuOpen });
  else setUi({ rail: !ui.rail });
}
const menuLabel = computed(() => (isMobile() ? "Abrir menu" : ui.rail ? "Expandir menu" : "Recolher menu"));
function toggleTheme() { setUi({ theme: isDark() ? "light" : "dark" }); }

watch(() => route.fullPath, () => { if (ui.menuOpen) setUi({ menuOpen: false }); });
watch(() => route.name, (n) => { if (n === "home") page.title = "Este servidor"; });

function toggleSearch() {
  if (!isMobile()) { document.getElementById("busca")?.focus(); return; }
  searchOpen.value = !searchOpen.value;
  if (searchOpen.value) nextTick(() => document.getElementById("busca")?.focus());
}

function runSearch() {
  const v = q.value.trim();
  if (v.length < 2) return;
  const src = route.name === "browse" ? route.params.source : undefined;
  router.push({ name: "search", query: { q: v, ...(src ? { source: src } : {}) } });
  searchOpen.value = false;
}

function onKey(e) {
  if (e.key === "/" && !/input|textarea|select/i.test(document.activeElement?.tagName || "") && !document.activeElement?.isContentEditable) {
    e.preventDefault();
    searchOpen.value = true;
    document.getElementById("busca")?.focus();
  }
}
function onExpired() { router.replace({ name: "login", query: { next: route.fullPath } }); }

let off = null;
onMounted(() => {
  window.addEventListener("keydown", onKey);
  window.addEventListener("lab:session-expired", onExpired);
  startUsbPolling();
  off = onTransferDone(() => { void refreshSources(); });
});
onBeforeUnmount(() => {
  window.removeEventListener("keydown", onKey);
  window.removeEventListener("lab:session-expired", onExpired);
  off && off();
});
watch(() => app.user, (u) => { if (u) { startUsbPolling(); void refreshUsb(); } });

const sidePath = (name) => routeFor(name, "/");
const usbPath = (d) => routeFor("Pendrives", "/" + d.name + "/");
</script>

<template>
  <svg width="0" height="0" style="position: absolute" aria-hidden="true">
    <defs>
      <linearGradient id="gdA" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#0bc7d3" /><stop offset=".55" stop-color="#7a76d4" /><stop offset="1" stop-color="#f31ed2" /></linearGradient>
      <linearGradient id="gdB" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#00c6cf" /><stop offset="1" stop-color="#065e9e" /></linearGradient>
      <linearGradient id="gdC" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ff9630" /><stop offset="1" stop-color="#ff1bca" /></linearGradient>
    </defs>
  </svg>

  <div v-if="!app.ready" class="splash"><Icon name="hdd" /></div>
  <router-view v-else-if="!logged" />
  <template v-else>
    <div class="app">
      <header class="top">
        <div class="brandcell"><Icon name="hdd" /><b>{{ appName }}</b></div>
        <div class="navbar">
          <button class="ib" :aria-label="menuLabel" :title="menuLabel" @click="toggleMenu"><Icon name="menu" /></button>
          <span class="crumbtop">{{ page.title }}</span>
          <span class="sp"></span>
          <form class="busca" :class="{ open: searchOpen }" role="search" @submit.prevent="runSearch">
            <button class="ib sbtn" type="button" aria-label="Buscar" title="Buscar (/)" @click="toggleSearch"><Icon name="search" /></button>
            <input id="busca" v-model="q" type="search" placeholder="Buscar arquivos…" aria-label="Buscar arquivos" autocomplete="off" @keydown.esc="searchOpen = false" />
          </form>
          <span v-if="usbOn && usbCount" class="ibw">
            <router-link class="ib" to="/" :title="usbCount + (usbCount === 1 ? ' pendrive ligado' : ' pendrives ligados')" :aria-label="usbCount + (usbCount === 1 ? ' pendrive ligado' : ' pendrives ligados')"><Icon name="usb" /></router-link>
            <span class="badge ok">{{ usbCount }}</span>
          </span>
          <span class="ibw">
            <button class="ib" aria-label="Transferências" title="Transferências" :aria-expanded="ui.transfersOpen" @click="setUi({ transfersOpen: !ui.transfersOpen })"><Icon name="transfer" /></button>
            <span v-if="activeCount" class="badge">{{ activeCount }}</span>
          </span>
          <button class="ib" aria-label="Alternar claro e escuro" title="Alternar claro e escuro" @click="toggleTheme"><Icon :name="isDark() ? 'sun' : 'moon'" /></button>
          <button class="ib" aria-label="Configurações" title="Configurações" @click="setUi({ settingsOpen: true })"><Icon name="gear" /></button>
        </div>
      </header>

      <TransfersPop />
      <TransferDock />

      <nav class="side" aria-label="Locais">
        <div class="scroll">
          <router-link class="nav" to="/" title="Este servidor" :aria-current="route.name === 'home' ? 'true' : undefined"><Icon name="home" /><span class="t">Este servidor</span></router-link>
          <div class="grp">Pastas</div>
          <router-link v-for="s in userSources()" :key="s.name" class="nav" :to="sidePath(s.name)" :title="s.name"
            :aria-current="route.params.source === s.name ? 'true' : undefined"><Icon :name="iconForSource(s.name)" /><span class="t">{{ s.name }}</span></router-link>
          <template v-if="hasUsbSource() && usbOn">
            <div class="grp">Pendrives</div>
            <router-link v-for="d in app.usb" :key="d.uuid" class="nav usb" :to="usbPath(d)" :title="d.name"
              :aria-current="route.params.source === 'Pendrives' && String(route.params.rest || '').startsWith(d.name + '/') ? 'true' : undefined">
              <Icon name="usb" /><span class="t">{{ d.name }}</span>
              <span v-if="d.ejectable" class="ej" role="button" tabindex="0" :aria-label="'Remover ' + d.name" title="Remover com segurança"
                @click.prevent.stop="eject(d)" @keydown.enter.prevent.stop="eject(d)"><Icon name="eject" /></span>
            </router-link>
            <div v-if="!app.usb.length" class="sem-usb"><Icon name="usb" /><span class="t">Nenhum ligado</span></div>
          </template>
        </div>
        <div class="foot">
          <button class="nav" title="Configurações" @click="setUi({ settingsOpen: true })"><Icon name="gear" /><span class="t">Configurações</span></button>
        </div>
      </nav>
      <div class="scrim" @click="setUi({ menuOpen: false })"></div>

      <main>
        <router-view />
        <footer v-if="status.show" class="statusbar" aria-live="polite">
          <span v-if="status.total > 0" class="sb-disk"><span class="sb-mini"><i :style="{ width: Math.min(100, Math.round(status.used / status.total * 100)) + '%' }"></i></span>{{ bytes(status.used) }} usados de {{ bytes(status.total) }}</span>
          <span v-else-if="status.label" class="sb-disk">{{ status.label }}</span>
          <span class="sp"></span>
          <template v-if="status.selCount">
            <span class="sb-n">de {{ status.count }} {{ status.noun }}</span><span class="sb-sep"></span>
            <span class="sb-sel"><b>{{ status.selCount }}</b> {{ status.selCount === 1 ? "selecionado" : "selecionados" }}<template v-if="status.selSize"> · <b>{{ bytes(status.selSize) }}</b></template></span>
          </template>
          <template v-else>
            <span class="sb-n"><b>{{ status.count }}</b> {{ status.count === 1 ? status.noun.replace(/s$/, "").replace("iten", "item") : status.noun }}</span>
            <template v-if="status.noun === 'itens'"><span class="sb-sep sb-opt"></span><span class="sb-opt">{{ status.folders }} {{ status.folders === 1 ? "pasta" : "pastas" }} · {{ status.files }} {{ status.files === 1 ? "arquivo" : "arquivos" }}</span></template>
            <template v-if="status.size"><span class="sb-sep"></span><span>soma <b>{{ bytes(status.size) }}</b></span></template>
          </template>
        </footer>
      </main>
    </div>
    <SettingsDrawer v-if="ui.settingsOpen" />
  </template>
  <DialogHost />
  <Toasts />
</template>
