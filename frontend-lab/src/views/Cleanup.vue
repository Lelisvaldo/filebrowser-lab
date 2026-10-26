<script setup>
import { ref, computed, watch, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as filesApi from "@/api/files";
import { app, page, hideStatus, toast, toastError, refreshSources, userSources, hasUsbSource, usbSourceName, canIn } from "@/store/app";
import { askConfirm } from "@/store/dialogs";
import { kindOf, iconKey, hueClass } from "@/lib/fileTypes";
import { iconForSource, hueOf } from "@/lib/sourceIcons";
import { bytes, pct } from "@/lib/format";
import { crumbs, routeFor } from "@/lib/paths";

const route = useRoute();
const router = useRouter();
const source = computed(() => String(route.params.source || ""));
const dirPath = computed(() => "/" + String(route.params.rest || ""));
const items = ref(null);
const sel = ref(new Set());
let mode0 = "map";
try { mode0 = localStorage.getItem("lab.limpezaView") === "list" ? "list" : "map"; } catch { /* sem storage */ }
const mode = ref(mode0);
const setMode = (m) => { mode.value = m; try { localStorage.setItem("lab.limpezaView", m); } catch { /* sem storage */ } };

const info = computed(() => app.sources[source.value] || {});
const crumbList = computed(() => crumbs(dirPath.value));
const folderName = computed(() => (dirPath.value === "/" ? source.value : dirPath.value.split("/").filter(Boolean).pop()));
const base = (p) => "/limpeza/" + encodeURIComponent(source.value) + routeFor(source.value, p).replace(/^\/f\/[^/]+/, "");
const canDelete = computed(() => canIn(source.value, "delete"));

const roots = computed(() => {
  const l = userSources().map((s) => s.name);
  if (hasUsbSource()) l.push(usbSourceName);
  return l;
});

async function load() {
  items.value = null;
  sel.value = new Set();
  if (!source.value) return;
  try {
    const d = await filesApi.list(source.value, dirPath.value);
    items.value = d.items.map((i) => ({ ...i, size: Number(i.size) || 0, dir: i.type === "directory" })).sort((a, b) => b.size - a.size);
  } catch (e) { items.value = []; toastError(e); }
}
watch([source, dirPath], () => {
  page.title = source.value ? "Limpeza · " + folderName.value : "Limpeza";
  void load();
}, { immediate: true });
onMounted(() => { hideStatus(); void refreshSources(); });

const total = computed(() => (items.value || []).reduce((s, i) => s + i.size, 0));
const share = (i) => (total.value ? (i.size / total.value) * 100 : 0);
const pctTxt = (i) => { const p = share(i); return p >= 1 ? Math.round(p) + "%" : p > 0 ? "<1%" : ""; };
const maxSize = computed(() => Math.max(1, ...(items.value || []).map((i) => i.size)));

const selItems = computed(() => (items.value || []).filter((i) => sel.value.has(i.path)));
const selSize = computed(() => selItems.value.reduce((s, i) => s + i.size, 0));
function toggle(i) {
  const s = new Set(sel.value);
  if (s.has(i.path)) s.delete(i.path); else s.add(i.path);
  sel.value = s;
}
function open(i) {
  if (i.dir) router.push(base(i.path));
  else router.push(routeFor(source.value, i.path).replace(/\/$/, ""));
}
function tap(i) { if (i.dir) open(i); else toggle(i); }

async function apagar() {
  const l = selItems.value;
  if (!l.length) return;
  const msg = l.length === 1 ? `"${l[0].name}" será apagado.` : `${l.length} itens serão apagados.`;
  const ok = await askConfirm({ title: "Apagar", message: msg + ` Libera ${bytes(selSize.value)}. Isso não pode ser desfeito.`, confirm: "Apagar", danger: true });
  if (!ok) return;
  try {
    await filesApi.remove(l.map((i) => ({ source: source.value, path: i.path })));
    toast("Apagado.");
    await load();
    void refreshSources();
  } catch (e) { toastError(e); }
}

// Mapa (blocos proporcionais ao tamanho, algoritmo squarified) numa área 100 x 62.
const W = 100, H = 62;
const blocks = computed(() => {
  const all = (items.value || []).filter((i) => i.size > 0);
  if (!all.length) return [];
  const tot = total.value;
  const main = [], rest = [];
  all.forEach((i, n) => { (n < 28 && i.size / tot >= 0.008 ? main : rest).push(i); });
  const nodes = main.map((i) => ({ it: i, v: i.size }));
  const rsize = rest.reduce((s, i) => s + i.size, 0);
  if (rsize > 0) nodes.push({ it: { name: `outros (${rest.length})`, size: rsize, path: "", rest: true }, v: rsize });
  const area = nodes.map((n) => ({ ...n, a: (n.v / tot) * W * H }));
  const out = [];
  let x = 0, y = 0, w = W, h = H, row = [];
  const sum = (r) => r.reduce((s, n) => s + n.a, 0);
  const worst = (r, side) => { const s = sum(r); const mx = Math.max(...r.map((n) => n.a)); const mn = Math.min(...r.map((n) => n.a)); return Math.max((side * side * mx) / (s * s), (s * s) / (side * side * mn)); };
  const flush = () => {
    const s = sum(row);
    if (w >= h) { const rw = s / h; let cy = y; for (const n of row) { const rh = n.a / rw; out.push({ it: n.it, x, y: cy, w: rw, h: rh }); cy += rh; } x += rw; w -= rw; }
    else { const rh = s / w; let cx = x; for (const n of row) { const rw = n.a / rh; out.push({ it: n.it, x: cx, y, w: rw, h: rh }); cx += rw; } y += rh; h -= rh; }
    row = [];
  };
  for (const n of area) { const side = Math.min(w, h); if (row.length && worst([...row, n], side) > worst(row, side)) flush(); row.push(n); }
  if (row.length) flush();
  return out.map((b, n) => ({ ...b, n }));
});
const PAL = ["cyan", "violet", "magenta", "orange", "green", "blue", "pink", "yellow", "red"];
const blockStyle = (b) => {
  const tint = b.it.rest ? "var(--sub)" : `var(--c-${PAL[b.n % PAL.length]})`;
  return {
    left: b.x + "%", top: (b.y / H) * 100 + "%", width: b.w + "%", height: (b.h / H) * 100 + "%",
    background: `color-mix(in srgb, ${tint} ${b.it.rest ? 40 : 62}%, #1b1530)`,
  };
};
const label = (b) => b.w * b.h > 60;
const tiny = (b) => b.w * b.h > 25;
</script>

<template>
  <div class="page">
    <div class="wrap wide cleanup">
      <div class="phead">
        <h1>Limpeza</h1>
        <div v-if="source" class="crumbs">
          <router-link to="/limpeza">Limpeza</router-link><span>/</span>
          <router-link :to="'/limpeza/' + encodeURIComponent(source) + '/'">{{ source }}</router-link>
          <template v-for="c in crumbList" :key="c.path"><span>/</span><router-link :to="base(c.path)">{{ c.name }}</router-link></template>
        </div>
        <p v-else class="sub">Veja o que mais ocupa espaço e apague o que não precisa mais. Escolha por onde começar.</p>
      </div>

      <div v-if="!source" class="rootgrid">
        <router-link v-for="n in roots" :key="n" class="rootcard" :to="'/limpeza/' + encodeURIComponent(n) + '/'">
          <Icon :name="iconForSource(n)" :class="hueOf(n)" />
          <span class="rn">{{ n }}</span>
          <template v-if="app.sources[n] && app.sources[n].total">
            <span class="bar"><i :style="{ width: pct(app.sources[n].used, app.sources[n].total) + '%' }"></i></span>
            <span class="rs">{{ bytes(app.sources[n].used) }} de {{ bytes(app.sources[n].total) }}</span>
          </template>
        </router-link>
      </div>

      <template v-else>
        <div class="cusage">
          <div class="cu-top"><b>{{ bytes(total) }}</b> nesta pasta<template v-if="info.total"> · disco {{ bytes(info.used) }} de {{ bytes(info.total) }} ({{ pct(info.used, info.total) }}%)</template></div>
          <div v-if="info.total" class="bar"><i :style="{ width: Math.max(1, pct(info.used, info.total)) + '%' }"></i></div>
        </div>

        <div class="tbar">
          <router-link class="ib" :to="routeFor(source, dirPath)" title="Abrir esta pasta nos arquivos" aria-label="Abrir esta pasta nos arquivos"><Icon name="folderopen" /></router-link>
          <span class="meta">{{ mode === "map" ? "Cada bloco é do tamanho que ocupa. Toque numa pasta para entrar." : "Do maior para o menor." }}</span>
          <span class="sp"></span>
          <div class="seg" role="group" aria-label="Modo">
            <button :aria-pressed="mode === 'map'" title="Mapa" aria-label="Mapa" @click="setMode('map')"><Icon name="grid" /></button>
            <button :aria-pressed="mode === 'list'" title="Lista" aria-label="Lista" @click="setMode('list')"><Icon name="list" /></button>
          </div>
        </div>

        <div v-if="items === null" class="vazio">Calculando…</div>
        <div v-else-if="!items.length" class="vazio vi"><Icon name="folder" /><span>Esta pasta está vazia.</span></div>

        <div v-else-if="mode === 'map'" class="tmap">
          <div v-for="b in blocks" :key="b.it.path || 'rest'" class="tm" :class="{ sel: sel.has(b.it.path), rest: b.it.rest }" :style="blockStyle(b)"
            :title="b.it.name + ' · ' + bytes(b.it.size) + ' · ' + pctTxt(b.it)"
            :tabindex="b.it.rest ? -1 : 0" @click="!b.it.rest && tap(b.it)" @keydown.enter="!b.it.rest && tap(b.it)">
            <template v-if="tiny(b)">
              <span v-if="!b.it.rest" class="tc" role="button" :aria-label="'Selecionar ' + b.it.name" @click.stop="toggle(b.it)"><Icon :name="sel.has(b.it.path) ? 'check' : b.it.dir ? 'folder' : iconKey(kindOf(b.it))" /></span>
              <span v-if="label(b)" class="tn">{{ b.it.name }}</span>
              <span v-if="b.w * b.h > 140" class="ts">{{ bytes(b.it.size) }} · {{ pctTxt(b.it) }}</span>
            </template>
          </div>
        </div>

        <div v-else class="flist">
          <div v-for="i in items" :key="i.path" class="crow" :class="{ sel: sel.has(i.path) }" tabindex="0" @click="tap(i)" @keydown.enter="tap(i)">
            <button class="chk" type="button" :aria-label="'Selecionar ' + i.name" :aria-pressed="sel.has(i.path)" @click.stop="toggle(i)"><Icon :name="sel.has(i.path) ? 'check' : i.dir ? 'folder' : iconKey(kindOf(i))" :class="sel.has(i.path) ? '' : hueClass(i.dir ? 'folder' : kindOf(i), source)" /></button>
            <span class="n">{{ i.name }}</span>
            <div class="bar" :class="{ cheio: share(i) > 40 }"><i :style="{ width: Math.max(2, (i.size / maxSize) * 100) + '%' }"></i></div>
            <span class="m">{{ pctTxt(i) }}</span>
            <span class="s">{{ bytes(i.size) }}</span>
          </div>
        </div>

        <div v-if="selItems.length" class="selbar" role="region" aria-live="polite">
          <span><b>{{ selItems.length }}</b> {{ selItems.length === 1 ? "selecionado" : "selecionados" }} · libera <b>{{ bytes(selSize) }}</b></span>
          <span class="sp"></span>
          <button class="btn" type="button" @click="sel = new Set()">Limpar seleção</button>
          <button v-if="canDelete" class="btn danger" type="button" @click="apagar"><Icon name="trash" /> Apagar</button>
        </div>
      </template>
    </div>
  </div>
</template>
