<script setup>
// Documentos do Office de qualquer geração: o servidor converte para PDF (LibreOffice) e mostramos aqui.
import { ref, watch, onMounted, onBeforeUnmount } from "vue";
import * as officeApi from "@/api/office";
const props = defineProps({ source: String, path: String, name: String });
const state = ref("loading");
const url = ref("");
const msg = ref("");
async function load() {
  state.value = "loading";
  if (url.value) { URL.revokeObjectURL(url.value); url.value = ""; }
  try {
    const b = await officeApi.pdfBlob(props.source, props.path);
    url.value = URL.createObjectURL(b);
    state.value = "ok";
  } catch (e) { msg.value = (e && e.message) || ""; state.value = "err"; }
}
onMounted(load);
watch(() => props.path, load);
onBeforeUnmount(() => { if (url.value) URL.revokeObjectURL(url.value); });
</script>

<template>
  <div class="office-wrap">
    <div v-if="state === 'loading'" class="vw-msg">Preparando a visualização… a primeira vez pode levar alguns segundos.</div>
    <div v-else-if="state === 'err'" class="vw-msg">Não foi possível abrir este documento. {{ msg }} Baixe para abrir no computador.</div>
    <iframe v-else :src="url" class="vw-pdf" :title="name"></iframe>
  </div>
</template>
