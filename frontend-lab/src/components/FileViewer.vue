<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import Icon from "./Icon.vue";
import * as filesApi from "@/api/files";
import { kindOf, isEditable, iconKey } from "@/lib/fileTypes";
import { bytes, date } from "@/lib/format";
import { toast, toastError } from "@/store/app";
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

const src = computed(() => {
  if (kind.value === "video" || kind.value === "audio") return filesApi.streamUrl(props.source, props.item.path, token.value);
  if (kind.value === "image" || kind.value === "pdf") return filesApi.viewUrl(props.source, props.item.path, token.value);
  return "";
});

// texto
const text = ref("");
const original = ref("");
const loading = ref(false);
const tooBig = ref(false);
const mdPreview = ref(true);
const saving = ref(false);
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
      <button v-if="prev" class="ib" aria-label="Anterior" title="Anterior (←)" @click="emit('go', prev)"><Icon name="left" /></button>
      <button v-if="next" class="ib" aria-label="Próximo" title="Próximo (→)" @click="emit('go', next)"><Icon name="right" /></button>
      <button class="ib" aria-label="Compartilhar" title="Compartilhar" @click="emit('share', item)"><Icon name="share" /></button>
      <button v-if="canDownload" class="ib" aria-label="Baixar" title="Baixar" @click="download"><Icon name="download" /></button>
      <button v-if="canModify" class="ib" aria-label="Apagar" title="Apagar" @click="emit('delete', item)"><Icon name="trash" /></button>
    </header>

    <div class="vw-body">
      <img v-if="kind === 'image'" :src="src" :alt="item.name" class="vw-img" />
      <video v-else-if="kind === 'video'" :key="src" :src="src" class="vw-video" controls autoplay playsinline></video>
      <div v-else-if="kind === 'audio'" class="vw-audio"><div class="vw-ico"><Icon name="music" /></div><b>{{ item.name }}</b><audio :key="src" :src="src" controls autoplay></audio></div>
      <iframe v-else-if="kind === 'pdf'" :src="src" class="vw-pdf" :title="item.name"></iframe>
      <div v-else-if="isEditable(kind) && !tooBig" class="vw-text">
        <div v-if="loading" class="vw-msg">Carregando…</div>
        <div v-else-if="isMd && mdPreview" class="md" v-html="html"></div>
        <textarea v-else v-model="text" class="vw-area" spellcheck="false" :readonly="!canModify" aria-label="Conteúdo do arquivo"></textarea>
      </div>
      <div v-else class="vw-other">
        <div class="vw-ico"><Icon :name="iconKey(kind)" /></div>
        <b>{{ item.name }}</b>
        <span>{{ tooBig ? "Arquivo grande demais para editar aqui." : "Sem visualização para este tipo de arquivo." }}</span>
        <button v-if="canDownload" class="btn pri" @click="download"><Icon name="download" /><span>Baixar ({{ bytes(item.size) }})</span></button>
      </div>
    </div>
  </div>
</template>
