<script setup>
// Prévia de documentos do Word (.docx) feita no próprio navegador.
import { ref, watch, onMounted } from "vue";
const props = defineProps({ url: String });
const box = ref(null);
const state = ref("loading");
async function render() {
  state.value = "loading";
  try {
    const [res, lib] = await Promise.all([fetch(props.url, { credentials: "same-origin" }), import("docx-preview")]);
    if (!res.ok) throw new Error(String(res.status));
    const blob = await res.blob();
    box.value.innerHTML = "";
    await lib.renderAsync(blob, box.value, null, { inWrapper: true, ignoreWidth: false, ignoreHeight: true, breakPages: true, useBase64URL: true });
    state.value = "ok";
  } catch (e) { state.value = "err"; }
}
onMounted(render);
watch(() => props.url, render);
</script>

<template>
  <div class="docx-wrap">
    <div v-if="state === 'loading'" class="vw-msg">Abrindo documento…</div>
    <div v-else-if="state === 'err'" class="vw-msg">Não foi possível mostrar este documento. Baixe para abrir no computador.</div>
    <div ref="box" class="docx-box" :hidden="state !== 'ok'"></div>
  </div>
</template>
