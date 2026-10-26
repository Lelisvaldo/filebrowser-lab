<script setup>
import { computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import Donut from "@/components/Donut.vue";
import TransferRows from "@/components/TransferRows.vue";
import { hideStatus, app, ui, setUi, page, userSources, hasUsbSource, eject, ejecting, refreshSources, refreshUsb } from "@/store/app";
import { visible } from "@/store/transfers";
import { bytes, pct } from "@/lib/format";
import { iconForSource, hueOf } from "@/lib/sourceIcons";
import { routeFor } from "@/lib/paths";

const router = useRouter();
onMounted(() => { hideStatus(); page.title = "Este servidor"; void refreshSources(); void refreshUsb(); });

const folders = computed(() => userSources().map((s, i) => {
  const info = app.sources[s.name] || {};
  const total = Number(info.total) || 0;
  const used = Number(info.used) || 0;
  return { name: s.name, i, used, total, pct: pct(used, total), status: info.status, icon: iconForSource(s.name) };
}));

const usbs = computed(() => (app.usb || []).map((d) => {
  const used = Math.max(0, (d.size || 0) - (d.free ?? d.size ?? 0));
  return { ...d, used, pct: d.mounted ? pct(used, d.size) : 0 };
}));
const showUsb = computed(() => app.usb !== null && hasUsbSource());
const nActive = computed(() => visible().length);
const statusText = (s) => (s === "ready" ? "Indexada" : s === "indexing" ? "Indexando…" : "Pasta");
const open = (name) => router.push(routeFor(name, "/"));
const openUsb = (d) => router.push(routeFor("Pendrives", "/" + d.name + "/"));
</script>

<template>
  <div class="page">
    <div class="wrap">
      <div class="phead"><h1>Este servidor</h1><div class="crumbs"><span>Início</span></div></div>

      <section v-if="nActive" class="th">
        <button class="thb" type="button" :aria-expanded="!ui.tcomp" title="Compactar ou expandir" @click="setUi({ tcomp: !ui.tcomp })">
          <span class="tt"><span>Transferências em andamento</span><small>· {{ nActive }}</small></span><Icon name="down2" />
        </button>
        <TransferRows :compact="ui.tcomp" />
      </section>

      <template v-if="showUsb">
        <h2>Pendrives <small v-if="usbs.length">· {{ usbs.length }} ligado{{ usbs.length > 1 ? "s" : "" }}</small></h2>
        <div class="tiles" id="usbs">
          <div v-for="d in usbs" :key="d.uuid" class="tile cp usbc" :class="{ go: d.mounted, 'usb-down': !d.mounted }" tabindex="0" role="button" @click="d.mounted && openUsb(d)" @keydown.enter="d.mounted && openUsb(d)">
            <div class="row" style="gap: 10px">
              <div class="ico sm usb"><Icon name="usb" /></div>
              <div style="min-width: 0; flex: 1"><div class="nm">{{ d.name }}</div><div v-if="d.mounted" class="path" :title="d.mountpoint + ' · ' + d.filesystem">{{ d.mountpoint }}</div><div v-else class="path off">Desmontado: tire e espete de novo</div></div>
              <Donut v-if="d.mounted" mini :pct="d.pct" :index="1" :warn="d.pct > 85" hue="hue h-green" />
              <button v-if="d.ejectable" class="ib ejb" :class="{ busy: ejecting[d.uuid] }" :disabled="ejecting[d.uuid]" :aria-label="'Remover ' + d.name + ' com segurança'" title="Remover com segurança" @click.stop="eject(d)"><Icon name="eject" /></button>
            </div>
            <div v-if="d.mounted" class="cpm">
              <span v-if="ejecting[d.uuid]" style="color: var(--warn); font-weight: 500">Removendo… não tire o pendrive</span>
              <span v-else><b>{{ bytes(d.used) }}</b> usados</span>
              <span>{{ bytes(d.size) }}</span>
            </div>
          </div>
          <div v-if="!usbs.length" class="vazio vi usb-off"><Icon name="usb" /><span>Nenhum pendrive ligado.</span></div>
        </div>
      </template>

      <h2>Pastas</h2>
      <div class="tiles" id="pastas">
        <div v-for="f in folders" :key="f.name" class="tile cp go" tabindex="0" role="button" @click="open(f.name)" @keydown.enter="open(f.name)">
          <div class="row" style="gap: 10px">
            <div class="ico sm" :class="hueOf(f.name)"><Icon :name="f.icon" /></div>
            <div style="min-width: 0; flex: 1"><div class="nm">{{ f.name }}</div><div class="path">{{ statusText(f.status) }}</div></div>
            <Donut mini :pct="f.pct" :index="f.i" :hue="hueOf(f.name)" />
          </div>
          <div class="cpm">
            <span v-if="f.total > 0"><b>{{ bytes(f.used) }}</b> usados</span><span v-else>calculando…</span>
            <span v-if="f.total > 0">{{ bytes(f.total) }}</span>
          </div>
        </div>
        <div v-if="!folders.length" class="vazio">Nenhuma pasta disponível para este usuário.</div>
      </div>
    </div>
  </div>
</template>
