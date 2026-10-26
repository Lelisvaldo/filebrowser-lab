<script setup>
import { ref, computed } from "vue";
import { useRoute } from "vue-router";
import Icon from "./Icon.vue";
import TransferRows from "./TransferRows.vue";
import { visible, clearFinished } from "@/store/transfers";
import { ui, setUi } from "@/store/app";

const route = useRoute();
const open = ref(true);
const n = computed(() => visible().filter((t) => ["queued", "running", "paused"].includes(t.status)).length);
const show = computed(() => visible().length > 0 && route.name !== "home" && !ui.transfersOpen);
const pc = computed(() => {
  const l = visible();
  const total = l.reduce((s, t) => s + t.size, 0);
  return total ? Math.floor((l.reduce((s, t) => s + t.loaded, 0) / total) * 100) : 0;
});
</script>

<template>
  <div v-if="show" class="dock" role="region" aria-label="Transferências em andamento">
    <button class="dock-h" type="button" :aria-expanded="open" @click="open = !open">
      <span class="tt"><b>Transferências</b><small>{{ n ? "· " + n + " em andamento" : "· concluídas" }}</small></span>
      <span class="dock-pc">{{ pc }}%</span>
      <Icon name="down2" />
    </button>
    <div v-show="open" class="dock-b"><TransferRows compact /></div>
    <div v-if="!n" class="dock-f"><button class="btn mini-btn" type="button" @click="clearFinished()">Dispensar</button></div>
  </div>
</template>
