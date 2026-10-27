<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import Icon from "./Icon.vue";
import CodeEditor from "./CodeEditor.vue";
import DocxView from "./DocxView.vue";
import SheetView from "./SheetView.vue";
import Model3D from "./Model3D.vue";
import ArchiveBrowser from "./ArchiveBrowser.vue";
import { toVtt } from "@/lib/subtitles";
import * as filesApi from "@/api/files";
import { kindOf, isEditable, iconKey } from "@/lib/fileTypes";
import { bytes, date } from "@/lib/format";
import { toast, toastError, isMobile } from "@/store/app";
import { noteDownload } from "@/store/transfers";

const props = defineProps({
  source: String,
  item: Object,          // arquivo aberto
  siblings: Array,       // arquivos da pasta (para anterior/próximo)
  canModify: Boolean,
  canDownload: Boolean,
});
const emit = defineEmits(["close", "go", "saved", "delete", "share"]);

const kind = computed(() => kindOf(props.item));
const token = computed(() => props.item.viewToken || "");
const mediaList = computed(() => (props.siblings || []).filter((s) => ["image", "video", "audio"].includes(kindOf(s))));
const idx = computed(() => mediaList.value.findIndex((s) => s.path === props.item.path));
const prev = computed(() => (["image", "video", "audio"].includes(kind.value) && idx.value > 0 ? mediaList.value[idx.value - 1] : null));
const next = computed(() => (["image", "video", "audio"].includes(kind.value) && idx.value >= 0 && idx.value < mediaList.value.length - 1 ? mediaList.value[idx.value + 1] : null));

const queue = computed(() => (props.siblings || []).filter((s) => kindOf(s) === kind.value));
const qIdx = computed(() => queue.value.findIndex((s) => s.path === props.item.path));
const nextSame = computed(() => (qIdx.value >= 0 && qIdx.value < queue.value.length - 1 ? queue.value[qIdx.value + 1] : null));
const isPlayer = computed(() => kind.value === "video" || kind.value === "audio");
let auto0 = true;
try { auto0 = localStorage.getItem("lab.autoNext") !== "0"; } catch { /* sem storage */ }
const autoNext = ref(auto0);
const setAuto = (v) => { autoNext.value = v; try { localStorage.setItem("lab.autoNext", v ? "1" : "0"); } catch { /* sem storage */ } };
const qOpen = ref(false);
const onEnded = () => { if (autoNext.value && nextSame.value) emit("go", nextSame.value); };
const goQ = (it) => { if (it.path !== props.item.path) emit("go", it); };

// legendas (vídeo): as externas (.srt, .vtt, .ass) e as embutidas viram faixas do player
const tracks = ref([]);
async function loadSubs() {
  tracks.value.forEach((t) => URL.revokeObjectURL(t.url));
  tracks.value = [];
  if (kind.value !== "video") return;
  const path = props.item.path;
  try {
    const d = await filesApi.mediaMeta(props.source, path);
    const list = Array.isArray(d.subtitles) ? d.subtitles : [];
    const out = [];
    for (const [i, t] of list.entries()) {
      try {
        const res = await fetch(filesApi.subtitleUrl(props.source, path, t.name, t.embedded), { credentials: "same-origin" });
        if (!res.ok) continue;
        const vtt = toVtt(await res.text());
        out.push({ url: URL.createObjectURL(new Blob([vtt], { type: "text/vtt" })), label: t.title || t.language || t.name, srclang: (t.srclang || t.language || "und").slice(0, 8), def: i === 0 });
      } catch { /* legenda com problema: segue sem ela */ }
    }
    if (props.item.path === path) tracks.value = out;
  } catch { /* sem metadados: sem legendas */ }
}
watch(() => props.item.path, loadSubs, { immediate: true });
onBeforeUnmount(() => tracks.value.forEach((t) => URL.revokeObjectURL(t.url)));

const src = computed(() => {
  if (kind.value === "video" || kind.value === "audio") return filesApi.streamUrl(props.source, props.item.path, token.value);
  if (["image", "pdf", "doc", "table", "model"].includes(kind.value)) return filesApi.viewUrl(props.source, props.item.path, token.value);
  return "";
});

// texto
const text = ref("");
const original = ref("");
const loading = ref(false);
const tooBig = ref(false);
const mdPreview = ref(true);
const saving = ref(false);
const mobile = isMobile();
const dirty = computed(() => text.value !== original.value);
const isMd = computed(() => /\.(md|markdown)$/i.test(props.item.name));
const html = computed(() => DOMPurify.sanitize(marked.parse(text.value || "")));

async function loadText() {
  text.value = original.value = "";
  tooBig.value = false;
  if (!isEditable(kind.value)) return;
  if (props.item.size > 3 * 1024 * 1024) { tooBig.value = true; return; }
  loading.value = true;
  try {
    const d = await filesApi.info(props.source, props.item.path, true);
    text.value = original.value = d.content ?? "";
  } catch (e) { toastError(e); } finally { loading.value = false; }
}
watch(() => props.item.path, loadText, { immediate: true });

async function save() {
  saving.value = true;
  try {
    await filesApi.writeText(props.source, props.item.path, text.value);
    original.value = text.value;
    toast("Arquivo salvo.");
    emit("saved");
  } catch (e) { toastError(e); } finally { saving.value = false; }
}

function close() {
  if (dirty.value && !window.confirm("Há alterações não salvas. Sair mesmo assim?")) return;
  emit("close");
}
function download() {
  const a = document.createElement("a");
  a.href = filesApi.downloadUrl(props.source, [props.item.path]);
  a.download = props.item.name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  noteDownload(props.item.name, props.item.size);
}
function onKey(e) {
  if (e.target.closest && e.target.closest(".monaco-editor") && e.key !== "Escape") return;
  if (/input|textarea/i.test(e.target.tagName) && e.key !== "Escape") {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s" && props.canModify && dirty.value) { e.preventDefault(); save(); }
    return;
  }
  if (e.key === "Escape") close();
  else if (e.key === "ArrowLeft" && prev.value) emit("go", prev.value);
  else if (e.key === "ArrowRight" && next.value) emit("go", next.value);
}
onMounted(() => document.addEventListener("keydown", onKey));
onBeforeUnmount(() => document.removeEventListener("keydown", onKey));
</script>

<template>
  <div class="viewer-ov" role="dialog" aria-modal="true" :aria-label="item.name">
    <header class="vw-top">
      <button class="ib" aria-label="Fechar" title="Fechar (Esc)" @click="close"><Icon name="back" /></button>
      <div class="vw-name"><b>{{ item.name }}</b><small>{{ bytes(item.size) }} · {{ date(item.modified) }}</small></div>
      <span class="sp"></span>
      <template v-if="isEditable(kind) && !tooBig">
        <button v-if="isMd" class="ib" :title="mdPreview ? 'Editar' : 'Ver formatado'" :aria-label="mdPreview ? 'Editar' : 'Ver formatado'" @click="mdPreview = !mdPreview"><Icon :name="mdPreview ? 'edit' : 'eye'" /></button>
        <button v-if="canModify" class="btn pri sm" title="Salvar (Ctrl+S)" aria-label="Salvar" :disabled="!dirty || saving" @click="save"><Icon name="save" /><span class="lbl">{{ saving ? "Salvando…" : "Salvar" }}</span></button>
      </template>
      <button v-if="isPlayer && queue.length > 1" class="ib" aria-label="Fila de reprodução" title="Fila de reprodução" :aria-pressed="qOpen" @click="qOpen = !qOpen"><Icon name="list" /></button>
      <button v-if="prev" class="ib" aria-label="Anterior" title="Anterior (←)" @click="emit('go', prev)"><Icon name="left" /></button>
      <button v-if="next" class="ib" aria-label="Próximo" title="Próximo (→)" @click="emit('go', next)"><Icon name="right" /></button>
      <button class="ib" aria-label="Compartilhar" title="Compartilhar" @click="emit('share', item)"><Icon name="share" /></button>
      <button v-if="canDownload" class="ib" aria-label="Baixar" title="Baixar" @click="download"><Icon name="download" /></button>
      <button v-if="canModify" class="ib" aria-label="Apagar" title="Apagar" @click="emit('delete', item)"><Icon name="trash" /></button>
    </header>

    <div class="vw-body">
      <img v-if="kind === 'image'" :src="src" :alt="item.name" class="vw-img" />
      <video v-else-if="kind === 'video'" :key="src + tracks.length" :src="src" class="vw-video" controls autoplay playsinline crossorigin="use-credentials" @ended="onEnded"><track v-for="t in tracks" :key="t.url" kind="subtitles" :src="t.url" :label="t.label" :srclang="t.srclang" :default="t.def" /></video>
      <div v-else-if="kind === 'audio'" class="vw-audio"><div class="vw-ico"><Icon name="music" /></div><b>{{ item.name }}</b><audio :key="src" :src="src" controls autoplay @ended="onEnded"></audio></div>
      <ArchiveBrowser v-else-if="kind === 'zip'" :key="item.path" :source="source" :item="item" :can-download="canDownload" />
      <DocxView v-else-if="kind === 'doc'" :url="src" />
      <SheetView v-else-if="kind === 'table'" :url="src" />
      <Model3D v-else-if="kind === 'model'" :url="src" :name="item.name" />
      <iframe v-else-if="kind === 'pdf'" :src="src" class="vw-pdf" :title="item.name"></iframe>
      <div v-else-if="isEditable(kind) && !tooBig" class="vw-text">
        <div v-if="loading" class="vw-msg">Carregando…</div>
        <div v-else-if="isMd && mdPreview" class="md" v-html="html"></div>
        <CodeEditor v-else-if="!mobile" v-model="text" :name="item.name" :readonly="!canModify" @save="save" />
        <textarea v-else v-model="text" class="vw-area" spellcheck="false" :readonly="!canModify" aria-label="Conteúdo do arquivo"></textarea>
      </div>
      <div v-else class="vw-other">
        <div class="vw-ico"><Icon :name="iconKey(kind)" /></div>
        <b>{{ item.name }}</b>
        <span>{{ tooBig ? "Arquivo grande demais para editar aqui." : "Sem visualização para este tipo de arquivo." }}</span>
        <button v-if="canDownload" class="btn pri" @click="download"><Icon name="download" /><span>Baixar ({{ bytes(item.size) }})</span></button>
      </div>
      <aside v-if="qOpen && isPlayer" class="vw-queue" aria-label="Fila de reprodução">
        <div class="vq-head"><b>Fila</b><small>{{ qIdx + 1 }} de {{ queue.length }}</small></div>
        <label class="vq-auto"><input type="checkbox" :checked="autoNext" @change="setAuto($event.target.checked)" /> Tocar o próximo ao terminar</label>
        <button v-for="q in queue" :key="q.path" class="vq-item" :class="{ cur: q.path === item.path }" @click="goQ(q)"><Icon :name="iconKey(kind)" /><span>{{ q.name }}</span></button>
      </aside>
    </div>
  </div>
</template>
