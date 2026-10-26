<script setup>
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({ x: Number, y: Number, entries: Array });
const emit = defineEmits(["pick", "close"]);
const el = ref(null);
const pos = ref({ left: props.x + "px", top: props.y + "px" });

async function place() {
  await nextTick();
  const r = el.value?.getBoundingClientRect();
  if (!r) return;
  const left = Math.max(8, Math.min(props.x, window.innerWidth - r.width - 8));
  const top = Math.max(8, Math.min(props.y, window.innerHeight - r.height - 8));
  pos.value = { left: left + "px", top: top + "px" };
  el.value.querySelector("button")?.focus();
}
watch(() => [props.x, props.y], place);
const away = (e) => { if (el.value && !el.value.contains(e.target)) emit("close"); };
const esc = (e) => { if (e.key === "Escape") emit("close"); };
onMounted(() => { place(); setTimeout(() => document.addEventListener("mousedown", away), 0); document.addEventListener("keydown", esc); });
onBeforeUnmount(() => { document.removeEventListener("mousedown", away); document.removeEventListener("keydown", esc); });
</script>

<template>
  <div ref="el" class="ctx" role="menu" :style="pos">
    <template v-for="(e, i) in entries" :key="i">
      <div v-if="e.sep" class="ctx-sep"></div>
      <button v-else class="ctx-i" :class="{ danger: e.danger }" role="menuitem" @click="emit('pick', e.id)"><Icon :name="e.icon" /><span>{{ e.label }}</span></button>
    </template>
  </div>
</template>
