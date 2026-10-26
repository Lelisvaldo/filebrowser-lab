<script setup>
import { computed } from "vue";
import TransferRows from "./TransferRows.vue";
import { visible, clearFinished } from "@/store/transfers";
import { ui, setUi } from "@/store/app";

const n = computed(() => visible().filter((t) => ["queued", "running", "paused"].includes(t.status)).length);
</script>

<template>
  <div v-if="ui.transfersOpen" class="pop" role="dialog" aria-label="Transferências em andamento" @keydown.esc="setUi({ transfersOpen: false })">
    <header>Transferências <span v-if="n" class="badge">{{ n }}</span>
      <span class="sp"></span><button class="btn mini-btn" type="button" @click="clearFinished()">Limpar concluídas</button>
    </header>
    <div style="padding: 10px; overflow-y: auto"><TransferRows /></div>
  </div>
</template>
