<script setup>
// Escolher ícone e cor de uma pasta.
import { ref } from "vue";
import Icon from "./Icon.vue";
import { STYLE_ICONS, STYLE_HUES, folderLook, setStyle, prefs, keyOf } from "@/lib/folderPrefs";

const props = defineProps({ source: String, path: String, name: String });
const emit = defineEmits(["close"]);
const cur = prefs.styles[keyOf(props.source, props.path)] || {};
const icon = ref(cur.icon || "");
const hue = ref(cur.hue || "");
const look = () => ({ icon: icon.value || "folder", cls: hue.value ? "hue h-" + hue.value : folderLook(props.source, "/").cls });
function apply() { setStyle(props.source, props.path, { icon: icon.value, hue: hue.value }); emit("close"); }
function reset() { setStyle(props.source, props.path, null); emit("close"); }
</script>

<template>
  <div class="fs-ov" @click.self="emit('close')" @keydown.esc="emit('close')">
    <div class="fs-box" role="dialog" aria-modal="true" aria-label="Personalizar pasta">
      <div class="fs-top"><span class="fs-prev"><Icon :name="look().icon" :class="look().cls" /></span><div><b>{{ name }}</b><small>Ícone e cor desta pasta (só neste navegador)</small></div></div>
      <div class="fs-lbl">Ícone</div>
      <div class="fs-grid"><button v-for="i in STYLE_ICONS" :key="i" class="fs-i" :class="{ on: (icon || 'folder') === i }" :aria-label="i" @click="icon = i === 'folder' ? '' : i"><Icon :name="i" /></button></div>
      <div class="fs-lbl">Cor</div>
      <div class="fs-grid"><button class="fs-c fs-none" :class="{ on: !hue }" aria-label="Cor da pasta raiz" title="Cor da pasta raiz" @click="hue = ''">∅</button><button v-for="h in STYLE_HUES" :key="h" class="fs-c" :class="['h-' + h, { on: hue === h }]" :aria-label="h" @click="hue = h"></button></div>
      <div class="fs-act"><button class="btn" @click="reset">Restaurar padrão</button><span class="sp"></span><button class="btn" @click="emit('close')">Cancelar</button><button class="btn pri" @click="apply">Aplicar</button></div>
    </div>
  </div>
</template>
