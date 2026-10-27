<script setup>
// Navegar e mexer dentro de um compactado (zip, 7z, tar, rar, iso…) como se fosse uma pasta.
import { ref, computed, onMounted } from "vue";
import Icon from "./Icon.vue";
import * as arc from "@/api/archive";
import { askText, askConfirm, pickFolder } from "@/store/dialogs";
import { toast, toastError, refreshSources } from "@/store/app";
import { kindOf, iconKey, hueClass } from "@/lib/fileTypes";
import { bytes, date } from "@/lib/format";
import { isMobile } from "@/store/app";

const props = defineProps({ source: String, item: Object, canDownload: Boolean });
const dir = ref("");
const entries = ref(null);
const editable = ref(false);
const encrypted = ref(false);
const err = ref("");
const pw = ref("");
const sel = ref(new Set());
const busy = ref(false);
const fileInput = ref(null);
const prev = ref(null); // { name, inner, kind, url, text, orig }

const parts = computed(() => dir.value.split("/").filter(Boolean));
const selected = computed(() => (entries.value || []).filter((e) => sel.value.has(e.path)));
const kids = (e) => ({ name: e.name, type: e.dir ? "directory" : "" });

async function load() {
  entries.value = null; err.value = ""; sel.value = new Set();
  try {
    const r = await arc.list(props.source, props.item.path, dir.value, pw.value);
    entries.value = r.items; editable.value = r.editable; encrypted.value = r.encrypted;
  } catch (e) {
    entries.value = [];
    const m = String(e.message || e);
    if (/password|wrong|encrypted/i.test(m)) await askPw(); else err.value = m;
  }
}
async function askPw() {
  const v = await askText({ title: "Compactado com senha", label: "Senha do arquivo", value: "", confirm: "Abrir", extra: { password: true } });
  if (v) { pw.value = v; await load(); } else err.value = "Este compactado precisa de senha.";
}
onMounted(load);
const go = (d) => { dir.value = d; void load(); };
function openEntry(e) { if (e.dir) go(e.path); else void preview(e); }
function toggle(e) { const s = new Set(sel.value); if (s.has(e.path)) s.delete(e.path); else s.add(e.path); sel.value = s; }

const TEXT_MAX = 2 * 1024 * 1024;
async function preview(e) {
  const k = kindOf(kids(e));
  const base = { name: e.name, inner: e.path, kind: k, url: "", text: "", orig: "", size: e.size };
  if (["image", "video", "audio", "pdf"].includes(k) && !pw.value) { prev.value = { ...base, url: arc.fileUrl(props.source, props.item.path, e.path) }; return; }
  if (["code", "text"].includes(k) && e.size <= TEXT_MAX) {
    try { const b = await arc.fileBlob(props.source, props.item.path, e.path, pw.value); const t = await b.text(); prev.value = { ...base, text: t, orig: t }; } catch (x) { toastError(x); }
    return;
  }
  prev.value = base;
}
async function savePrev() {
  busy.value = true;
  try { await arc.edit({ source: props.source, path: props.item.path, op: "save", inner: prev.value.inner, content: prev.value.text }, pw.value); prev.value.orig = prev.value.text; toast("Salvo dentro do compactado."); } catch (e) { toastError(e); } finally { busy.value = false; }
}
function downloadInner(e) {
  const a = document.createElement("a"); a.href = arc.fileUrl(props.source, props.item.path, e.path, true); a.download = e.name; document.body.appendChild(a); a.click(); a.remove();
}

async function run(fn, ok) { busy.value = true; try { await fn(); if (ok) toast(ok); await load(); void refreshSources(); } catch (e) { toastError(e); } finally { busy.value = false; } }
const base = () => ({ source: props.source, path: props.item.path });
async function mkdir() {
  const name = await askText({ title: "Nova pasta no compactado", label: "Nome", value: "", confirm: "Criar" });
  if (name) await run(() => arc.edit({ ...base(), op: "mkdir", dir: dir.value, name }, pw.value), "Pasta criada.");
}
async function del() {
  const l = selected.value; if (!l.length) return;
  if (!(await askConfirm({ title: "Apagar do compactado", message: l.length === 1 ? `"${l[0].name}" será apagado de dentro do arquivo.` : `${l.length} itens serão apagados de dentro do arquivo.`, confirm: "Apagar", danger: true }))) return;
  await run(() => arc.edit({ ...base(), op: "delete", items: l.map((e) => e.path) }, pw.value), "Apagado.");
}
async function rename() {
  const e = selected.value[0]; if (!e) return;
  const name = await askText({ title: "Renomear", label: "Novo nome", value: e.name, confirm: "Renomear" });
  if (!name || name === e.name) return;
  const to = (dir.value ? dir.value + "/" : "") + name;
  await run(() => arc.edit({ ...base(), op: "rename", from: e.path, to }, pw.value), "Renomeado.");
}
async function onFiles(ev) {
  const files = [...ev.target.files].map((f) => ({ file: f, rel: f.webkitRelativePath || f.name })); ev.target.value = "";
  if (files.length) await run(() => arc.add(props.source, props.item.path, dir.value, files, pw.value), `${files.length} arquivo(s) adicionados.`);
}
async function extractTo(items) {
  const dest = await pickFolder({ title: items ? "Extrair seleção para…" : "Extrair tudo para…", extra: { source: props.source, path: "/", verb: "Extrair" } });
  if (!dest) return;
  busy.value = true;
  try { await arc.extract({ source: props.source, path: props.item.path, toSource: dest.source, dest: dest.path, items: items || [], password: pw.value }); toast("Extraído."); void refreshSources(); } catch (e) { toastError(e); } finally { busy.value = false; }
}
const canEdit = computed(() => editable.value);
const prevDirty = computed(() => prev.value && prev.value.text !== prev.value.orig);
function closePrev() { if (prevDirty.value && !window.confirm("Há alterações não salvas. Sair mesmo assim?")) return; prev.value = null; }
</script>

<template>
  <div class="arc">
    <div class="arc-bar">
      <div class="arc-crumbs"><button type="button" @click="go('')"><Icon name="zip" /> {{ item.name }}</button><template v-for="(p, i) in parts" :key="i"><span>/</span><button type="button" @click="go(parts.slice(0, i + 1).join('/'))">{{ p }}</button></template></div>
      <span class="sp"></span>
      <template v-if="!selected.length">
        <button v-if="canEdit" class="btn" type="button" :disabled="busy" @click="fileInput.click()"><Icon name="upload" /><span class="lbl">Adicionar</span></button>
        <button v-if="canEdit" class="btn" type="button" :disabled="busy" @click="mkdir"><Icon name="foldernew" /><span class="lbl">Nova pasta</span></button>
        <button class="btn" type="button" :disabled="busy || !canDownload" @click="extractTo(null)"><Icon name="zip" /><span class="lbl">Extrair tudo</span></button>
      </template>
      <template v-else>
        <b>{{ selected.length }}</b>
        <button v-if="canDownload" class="btn" type="button" :disabled="busy" @click="extractTo(selected.map((e) => e.path))"><Icon name="zip" /><span class="lbl">Extrair</span></button>
        <button v-if="canEdit && selected.length === 1" class="btn" type="button" :disabled="busy" @click="rename"><Icon name="edit" /><span class="lbl">Renomear</span></button>
        <button v-if="canEdit" class="btn danger" type="button" :disabled="busy" @click="del"><Icon name="trash" /><span class="lbl">Apagar</span></button>
        <button class="ib" type="button" aria-label="Limpar seleção" @click="sel = new Set()"><Icon name="dismiss" /></button>
      </template>
      <input ref="fileInput" type="file" multiple hidden @change="onFiles" />
    </div>
    <div v-if="!canEdit && entries" class="arc-note">Somente leitura{{ /\.(zip|7z|tar)$/i.test(item.name) ? "" : " (este formato não permite alterar por dentro)" }}.</div>
    <div v-if="entries === null" class="vw-msg">Abrindo compactado…</div>
    <div v-else-if="err" class="vw-msg">{{ err }}</div>
    <div v-else-if="!entries.length" class="vw-msg">Pasta vazia.</div>
    <div v-else class="arc-list" role="table">
      <div v-for="e in entries" :key="e.path" class="arc-row" :class="{ sel: sel.has(e.path) }" tabindex="0" @click="openEntry(e)" @keydown.enter="openEntry(e)">
        <button class="chk" type="button" :aria-label="'Selecionar ' + e.name" @click.stop="toggle(e)"><Icon :name="sel.has(e.path) ? 'checkboxon' : e.dir ? 'folder' : iconKey(kindOf(kids(e)))" :class="sel.has(e.path) ? '' : e.dir ? 'hue h-orange' : hueClass(kindOf(kids(e)))" /></button>
        <span class="n">{{ e.name }}</span>
        <span class="m hide-s">{{ e.modified ? date(e.modified) : "" }}</span>
        <span class="s">{{ e.dir && !e.size ? "—" : bytes(e.size) }}</span>
        <button v-if="!e.dir && canDownload" class="ib" type="button" aria-label="Baixar" title="Baixar" @click.stop="downloadInner(e)"><Icon name="download" /></button><span v-else></span>
      </div>
    </div>

    <div v-if="prev" class="fs-ov arc-prev" @click.self="closePrev">
      <div class="fs-box arc-pbox" role="dialog" aria-modal="true" :aria-label="prev.name">
        <div class="arc-ptop"><b>{{ prev.name }}</b><small>{{ bytes(prev.size) }}</small><span class="sp"></span>
          <button v-if="canEdit && prev.kind && ['code','text'].includes(prev.kind) && prev.orig !== undefined && !prev.url" class="btn pri sm" :disabled="!prevDirty || busy" @click="savePrev"><Icon name="save" /><span class="lbl">Salvar</span></button>
          <button v-if="canDownload" class="ib" type="button" aria-label="Baixar" @click="downloadInner({ path: prev.inner, name: prev.name })"><Icon name="download" /></button>
          <button class="ib" type="button" aria-label="Fechar" @click="closePrev"><Icon name="dismiss" /></button></div>
        <img v-if="prev.kind === 'image' && prev.url" :src="prev.url" :alt="prev.name" class="arc-img" />
        <video v-else-if="prev.kind === 'video' && prev.url" :src="prev.url" class="arc-img" controls autoplay playsinline></video>
        <audio v-else-if="prev.kind === 'audio' && prev.url" :src="prev.url" controls autoplay></audio>
        <iframe v-else-if="prev.kind === 'pdf' && prev.url" :src="prev.url" class="arc-pdf" :title="prev.name"></iframe>
        <textarea v-else-if="['code','text'].includes(prev.kind) && !prev.url && prev.size <= 2 * 1024 * 1024" v-model="prev.text" class="vw-area arc-area" spellcheck="false" :readonly="!canEdit" aria-label="Conteúdo"></textarea>
        <div v-else class="vw-msg">Sem visualização para este tipo. Use Baixar.</div>
      </div>
    </div>
  </div>
</template>
