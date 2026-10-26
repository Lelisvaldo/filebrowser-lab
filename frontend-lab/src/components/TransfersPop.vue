<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from "vue";
import TransferRows from "./TransferRows.vue";
import { visible, clearFinished } from "@/store/transfers";
import { ui, setUi } from "@/store/app";

const el = ref(null);
function away(e) {
  if (!ui.transfersOpen) return;
  if (el.value && !el.value.contains(e.target) && !e.target.closest('[aria-label="Transferências"]')) setUi({ transfersOpen: false });
}
onMounted(() => document.addEventListener("mousedown", away));
onBeforeUnmount(() => document.removeEventListener("mousedown", away));
const finished = computed(() => visible().filter((t) => ["done", "error"].includes(t.status)).length);
const n = computed(() => visible().filter((t) => ["queued", "running", "paused"].includes(t.status)).length);
</script>

<template>
  <div v-if="ui.transfersOpen" ref="el" class="pop" role="dialog" aria-label="Transferências em andamento" @keydown.esc="setUi({ transfersOpen: false })">
    <header>Transferências <span v-if="n" class="badge">{{ n }}</span>
      <span class="sp"></span><button v-if="finished" class="btn mini-btn" type="button" @click="clearFinished()">Limpar concluídas</button>
    </header>
    <div style="padding: 10px; overflow-y: auto"><TransferRows /></div>
  </div>
</template>
