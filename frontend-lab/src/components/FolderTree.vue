<script setup>
// Árvore de pastas do menu lateral: carrega as subpastas só quando abre; aceita soltar itens para mover.
import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";
import { useRoute } from "vue-router";
import Icon from "./Icon.vue";
import * as filesApi from "@/api/files";
import { routeFor } from "@/lib/paths";
import { folderLook } from "@/lib/folderPrefs";
import { isLabDrag, dropInto } from "@/store/dnd";
import { isOpen, toggleOpen } from "@/lib/tree";

const props = defineProps({ source: String, path: String, depth: { type: Number, default: 1 } });
const route = useRoute();
const dirs = ref(null);
const over = ref("");
const collator = new Intl.Collator("pt-BR", { numeric: true, sensitivity: "base" });

async function load() {
  try {
    const d = await filesApi.list(props.source, props.path);
    dirs.value = d.items.filter((i) => i.type === "directory" && !i.hidden).sort((a, b) => collator.compare(a.name, b.name));
  } catch (_e) { dirs.value = []; }
}
onMounted(() => { void load(); window.addEventListener("lab:refresh", load); });
onBeforeUnmount(() => window.removeEventListener("lab:refresh", load));
watch(() => [props.source, props.path], load);

const here = computed(() => (route.name === "browse" ? "/" + String(route.params.rest || "") : ""));
const active = (d) => route.params.source === props.source && here.value.replace(/\/?$/, "/") === d.path;
function dragOver(d, e) { if (isLabDrag(e)) { e.preventDefault(); over.value = d.path; } }
async function drop(d, e) { e.preventDefault(); over.value = ""; await dropInto(e, props.source, d.path); }
</script>

<template>
  <div v-if="dirs && dirs.length" class="tree" role="group">
    <template v-for="d in dirs" :key="d.path">
      <div class="nav tnode" :class="{ dropT: over === d.path }" :style="{ paddingLeft: 8 + depth * 14 + 'px' }" @dragover="dragOver(d, $event)" @dragleave="over = ''" @drop="drop(d, $event)">
        <button class="tw" type="button" :aria-label="isOpen(source, d.path) ? 'Recolher' : 'Expandir'" :aria-expanded="isOpen(source, d.path)" @click.stop.prevent="toggleOpen(source, d.path)"><Icon name="chev" :class="{ open: isOpen(source, d.path) }" /></button>
        <router-link class="tl" :to="routeFor(source, d.path)" :title="d.name" :aria-current="active(d) ? 'true' : undefined"><Icon :name="folderLook(source, d.path).icon" :class="folderLook(source, d.path).cls" /><span class="t">{{ d.name }}</span></router-link>
      </div>
      <FolderTree v-if="isOpen(source, d.path)" :source="source" :path="d.path" :depth="depth + 1" />
    </template>
  </div>
</template>
