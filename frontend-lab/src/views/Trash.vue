<script setup>
import { ref, computed, onMounted } from "vue";
import Icon from "@/components/Icon.vue";
import * as trashApi from "@/api/trash";
import { toast, toastError, refreshSources } from "@/store/app";
import { trash } from "@/store/trash";
import { askConfirm } from "@/store/dialogs";
import { bytes, date } from "@/lib/format";
import { iconForSource, hueOf } from "@/lib/sourceIcons";
import { kindOf, iconKey, hueClass } from "@/lib/fileTypes";

const data = ref(null); // { items, days, admin, drive }
const drive = ref(null); // { items } (só administrador, com o Google Drive conectado)
const sel = ref(new Set()); // "src:id" do servidor
const dsel = ref(new Set()); // id do Drive
const busy = ref(false);

const key = (it) => it.source + ":" + it.id;
const items = computed(() => data.value?.items || []);
const days = computed(() => data.value?.days || 30);
const fmt = (unix) => date(new Date(unix * 1000).toISOString());
const left = (it) => { const d = Math.ceil((it.expiresAt * 1000 - Date.now()) / 86400000); return d <= 0 ? "vence hoje" : d === 1 ? "falta 1 dia" : `faltam ${d} dias`; };
const icon = (it) => (it.isDir ? "folder" : iconKey(kindOf({ name: it.name, type: "file" })));
const hue = (it) => (it.isDir ? hueOf(it.source) : hueClass(kindOf({ name: it.name, type: "file" })));
const where = (it) => it.source + (it.path.includes("/") ? it.path.replace(/\/[^/]*$/, "") || "/" : "");

async function load() {
  try { data.value = await trashApi.list(); trash.count = data.value.items.length; trash.days = data.value.days; } catch (e) { data.value = { items: [], days: 30 }; toastError(e); }
  sel.value = new Set();
  if (data.value?.drive) {
    try { drive.value = await trashApi.driveList(); } catch (e) { drive.value = { items: [] }; if (e.status !== 412) toastError(e); }
  } else drive.value = null;
  dsel.value = new Set();
}
onMounted(load);

const flip = (cur, k) => { const n = new Set(cur); n.has(k) ? n.delete(k) : n.add(k); return n; };
const toggleSel = (k) => { sel.value = flip(sel.value, k); };
const toggleD = (k) => { dsel.value = flip(dsel.value, k); };
const allOn = computed(() => items.value.length > 0 && sel.value.size === items.value.length);
const toggleAll = () => { sel.value = allOn.value ? new Set() : new Set(items.value.map(key)); };
const dAllOn = computed(() => (drive.value?.items?.length || 0) > 0 && dsel.value.size === drive.value.items.length);
const dToggleAll = () => { dsel.value = dAllOn.value ? new Set() : new Set(drive.value.items.map((x) => x.id)); };

const picked = () => items.value.filter((it) => sel.value.has(key(it))).map((it) => ({ source: it.source, id: it.id }));

function report(what, r, ok) {
  const fail = r?.failed?.length || 0;
  if (fail) toast(`${ok} ${what}; ${fail} não deu: ${r.failed[0].message}`, "err", 7000);
  else toast(`${ok} ${what}.`);
}

async function doRestore() {
  const l = picked(); if (!l.length || busy.value) return;
  busy.value = true;
  try { const r = await trashApi.restore(l); report(r.restored === 1 ? "item restaurado" : "itens restaurados", r, r.restored); await load(); void refreshSources(); } catch (e) { toastError(e); } finally { busy.value = false; }
}
async function doPurge() {
  const l = picked(); if (!l.length || busy.value) return;
  const ok = await askConfirm({ title: "Apagar de vez", message: `${l.length === 1 ? "Este item será apagado" : l.length + " itens serão apagados"} de vez. Não dá para recuperar.`, confirm: "Apagar de vez", danger: true });
  if (!ok) return;
  busy.value = true;
  try { const r = await trashApi.purge(l); report(r.purged === 1 ? "item apagado de vez" : "itens apagados de vez", r, r.purged); await load(); void refreshSources(); } catch (e) { toastError(e); } finally { busy.value = false; }
}
async function doEmpty() {
  if (!items.value.length || busy.value) return;
  const ok = await askConfirm({ title: "Esvaziar a lixeira", message: `${items.value.length === 1 ? "O único item será apagado" : "Todos os " + items.value.length + " itens serão apagados"} de vez${data.value?.admin ? " (de todos os usuários)" : ""}. Não dá para recuperar.`, confirm: "Esvaziar", danger: true });
  if (!ok) return;
  busy.value = true;
  try { const r = await trashApi.purgeAll(); report("itens apagados de vez", r, r.purged); await load(); void refreshSources(); } catch (e) { toastError(e); } finally { busy.value = false; }
}

const dPicked = () => [...dsel.value];
async function dRestore() {
  const l = dPicked(); if (!l.length || busy.value) return;
  busy.value = true;
  try { const r = await trashApi.driveRestore(l); report("restaurado(s) no Google Drive", r, r.restored); await load(); } catch (e) { toastError(e); } finally { busy.value = false; }
}
async function dPurge() {
  const l = dPicked(); if (!l.length || busy.value) return;
  const ok = await askConfirm({ title: "Apagar do Google Drive", message: `${l.length === 1 ? "Este item será apagado" : l.length + " itens serão apagados"} de vez do Google Drive. Não dá para recuperar.`, confirm: "Apagar de vez", danger: true });
  if (!ok) return;
  busy.value = true;
  try { const r = await trashApi.drivePurge(l); report("apagado(s) de vez do Google Drive", r, r.purged); await load(); } catch (e) { toastError(e); } finally { busy.value = false; }
}
async function dEmpty() {
  if (busy.value) return;
  const ok = await askConfirm({ title: "Esvaziar a lixeira do Google Drive", message: "Tudo que está na lixeira do Google Drive será apagado de vez, inclusive o que foi apagado pelo site ou pelo celular. Não dá para recuperar.", confirm: "Esvaziar", danger: true });
  if (!ok) return;
  busy.value = true;
  try { await trashApi.driveEmpty(); toast("Lixeira do Google Drive esvaziada."); await load(); } catch (e) { toastError(e); } finally { busy.value = false; }
}
</script>

<template>
  <div class="page">
    <div class="wrap">
      <div class="phead"><div class="ph-t hue h-red"><span class="ph-ico"><Icon name="trash" /></span><h1>Lixeira</h1></div>
        <div class="crumbs"><span>{{ data?.admin ? "todos os usuários" : "o que você apagou" }}</span></div></div>

      <section class="card">
        <div class="cardhead"><h2>Pastas do servidor</h2><button class="ib" aria-label="Atualizar" title="Atualizar" @click="load"><Icon name="refresh" /></button></div>
        <p class="tr-note">O que você apaga fica aqui por {{ days }} dias e depois some sozinho. Pendrives não têm lixeira (apagam direto). Nas nuvens, o item vai para a lixeira do próprio serviço.</p>
        <div v-if="data === null" class="vazio">Carregando…</div>
        <div v-else-if="!items.length" class="vazio">A lixeira está vazia.</div>
        <template v-else>
          <div class="tr-bar">
            <label class="chk"><input type="checkbox" :checked="allOn" @change="toggleAll" /> Selecionar tudo</label>
            <span class="sp" style="flex: 1"></span>
            <button class="btn" type="button" :disabled="!sel.size || busy" @click="doRestore"><Icon name="refresh" /> Restaurar</button>
            <button class="btn danger" type="button" :disabled="!sel.size || busy" @click="doPurge"><Icon name="trash" /> Apagar de vez</button>
            <button class="btn danger" type="button" :disabled="busy" @click="doEmpty">Esvaziar</button>
          </div>
          <div class="tbl">
            <label v-for="it in items" :key="key(it)" class="trow" :title="it.source + it.path">
              <input type="checkbox" :checked="sel.has(key(it))" @change="toggleSel(key(it))" />
              <span class="tr-ico"><Icon :name="icon(it)" :class="hue(it)" /></span>
              <div class="main"><b>{{ it.name }}</b><small>{{ where(it) }}<template v-if="data.admin"> · apagado por {{ it.by }}</template><template v-if="it.size >= 0"> · {{ bytes(it.size) }}</template></small></div>
              <span class="tr-when">{{ fmt(it.deletedAt) }}<br />{{ left(it) }}</span>
            </label>
          </div>
        </template>
      </section>

      <template v-if="drive">
        <div class="tr-sec"><Icon :name="iconForSource('Nuvens')" class="hue h-cyan" /><h2>Google Drive</h2></div>
        <section class="card">
          <p class="tr-note">É a lixeira do próprio Google Drive (o Drive a esvazia sozinho depois de um tempo). Só administradores veem esta parte.</p>
          <div v-if="!drive.items.length" class="vazio">A lixeira do Google Drive está vazia.</div>
          <template v-else>
            <div class="tr-bar">
              <label class="chk"><input type="checkbox" :checked="dAllOn" @change="dToggleAll" /> Selecionar tudo</label>
              <span class="sp" style="flex: 1"></span>
              <button class="btn" type="button" :disabled="!dsel.size || busy" @click="dRestore"><Icon name="refresh" /> Restaurar</button>
              <button class="btn danger" type="button" :disabled="!dsel.size || busy" @click="dPurge"><Icon name="trash" /> Apagar de vez</button>
              <button class="btn danger" type="button" :disabled="busy" @click="dEmpty">Esvaziar</button>
            </div>
            <div class="tbl">
              <label v-for="it in drive.items" :key="it.id" class="trow">
                <input type="checkbox" :checked="dsel.has(it.id)" @change="toggleD(it.id)" />
                <span class="tr-ico"><Icon :name="it.isDir ? 'folder' : iconKey(kindOf({ name: it.name, type: 'file' }))" :class="it.isDir ? 'hue h-cyan' : hueClass(kindOf({ name: it.name, type: 'file' }))" /></span>
                <div class="main"><b>{{ it.name }}</b><small><template v-if="it.size > 0">{{ bytes(it.size) }}</template></small></div>
                <span class="tr-when">{{ it.deletedAt ? date(it.deletedAt) : "" }}</span>
              </label>
            </div>
          </template>
        </section>
      </template>
    </div>
  </div>
</template>
